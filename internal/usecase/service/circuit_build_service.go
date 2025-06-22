package service

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// ---- CircuitBuilder --------------------------------------------------------

// CircuitBuilder は Circuit を生成するためのインターフェース。
type CircuitBuildService interface {
	// Build は新しい Circuit を生成してリポジトリに保存し、返す。
	Build(hops int) (*entity.Circuit, error)
}

// CircuitBuilder はリレー選択・鍵生成・Circuit 保存をまとめたドメインサービス。
type circuitBuildServiceImpl struct {
	rr     repository.RelayRepository
	cr     repository.CircuitRepository
	dialer CircuitDialer
	crypto CryptoService
}

func NewCircuitBuildService(rr repository.RelayRepository, cr repository.CircuitRepository, d CircuitDialer, c CryptoService) CircuitBuildService {
	return &circuitBuildServiceImpl{rr: rr, cr: cr, dialer: d, crypto: c}
}

// Build は新しい Circuit を生成してリポジトリに保存し、返す。
func (b *circuitBuildServiceImpl) Build(hops int) (*entity.Circuit, error) {
	if hops <= 0 {
		hops = 3
	}
	// 1. オンラインリスト取得
	relays, err := b.rr.AllOnline()
	if err != nil {
		return nil, fmt.Errorf("list relays: %w", err)
	}
	if len(relays) < hops {
		return nil, fmt.Errorf("not enough online relays (need %d)", hops)
	}

	// 2. 無作為に hops 個選出（重複なし）
	if err := secureShuffle(relays); err != nil {
		return nil, fmt.Errorf("shuffle relays: %w", err)
	}
	selected := relays[:hops]

	relayIDs := make([]value_object.RelayID, 0, hops)
	keys := make([]value_object.AESKey, hops)
	nonces := make([]value_object.Nonce, hops)

	for _, r := range selected {
		relayIDs = append(relayIDs, r.ID())
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	// 3. CircuitID 生成
	cid := value_object.NewCircuitID()

	// --- build circuit over the network ---
	conn, err := b.dialer.Dial(selected[0].Endpoint().String())
	if err != nil {
		return nil, err
	}

	for i := 0; i < hops; i++ {
		next := ""
		if i+1 < hops {
			next = selected[i+1].Endpoint().String()
		}
		cliPriv, cliPub, err := b.crypto.X25519Generate()
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		var pubArr [32]byte
		copy(pubArr[:], cliPub)
		payload, err := value_object.EncodeExtendPayload(&value_object.ExtendPayload{
			NextHop:   next,
			ClientPub: pubArr,
		})
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		cell := entity.Cell{CircID: cid, StreamID: 0, Data: payload}
		if err := b.dialer.SendCell(conn, cell); err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		resp, err := b.dialer.WaitCreated(conn)
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		created, err := value_object.DecodeCreatedPayload(resp)
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		secret, err := b.crypto.X25519Shared(cliPriv, created.RelayPub[:])
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		key, nonce, err := b.crypto.DeriveKeyNonce(secret)
		if err != nil {
			_ = b.dialer.SendDestroy(conn, cid)
			conn.Close()
			return nil, err
		}
		keys[i] = key
		nonces[i] = nonce
	}

	circuit, err := entity.NewCircuit(cid, relayIDs, keys, nonces, priv)
	if err != nil {
		_ = b.dialer.SendDestroy(conn, cid)
		conn.Close()
		return nil, err
	}
	circuit.SetConn(0, conn)

	// 5. 保存
	if err := b.cr.Save(circuit); err != nil {
		_ = b.dialer.SendDestroy(conn, cid)
		conn.Close()
		return nil, fmt.Errorf("save circuit: %w", err)
	}

	return circuit, nil
}

func secureShuffle[T any](xs []T) error {
	for i := len(xs) - 1; i > 0; i-- {
		var b [2]byte
		if _, err := rand.Read(b[:]); err != nil { // crypto/rand
			return err
		}
		j := int(binary.BigEndian.Uint16(b[:])) % (i + 1)
		xs[i], xs[j] = xs[j], xs[i]
	}
	return nil
}
