// file: internal/usecase/build_circuit_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"ikedadada/go-ptor/shared/domain/aggregate"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
	"net"
	"time"
)

// ---------- DTO ----------

// BuildCircuitInput はユーザーが指定できるパラメータ
type BuildCircuitInput struct {
	Hops        int    // 省略時はデフォルト (3)
	ExitRelayID string // 任意。指定時は最終 hop をこのリレーに固定
}

// BuildCircuitOutput は UI / API に返すレスポンス
type BuildCircuitOutput struct {
	CircuitID vo.CircuitID `json:"circuit_id"`
}

// BuildCircuitUseCase creates new circuits according to the input parameters.
type BuildCircuitUseCase interface {
	Handle(input BuildCircuitInput) (BuildCircuitOutput, error)
}

// ---------- 実装 ----------

type buildCircuitUseCaseImpl struct {
	rRepo repository.RelayRepository
	cRepo repository.CircuitRepository
	cbSvc service.CircuitBuildService
	cSvc  service.CryptoService
	peSvc service.PayloadEncodingService
}

// NewBuildCircuitUseCase creates a use case for building circuits.
func NewBuildCircuitUseCase(rRepo repository.RelayRepository, cRepo repository.CircuitRepository, cbSvc service.CircuitBuildService, cSvc service.CryptoService, peSvc service.PayloadEncodingService) BuildCircuitUseCase {
	return &buildCircuitUseCaseImpl{rRepo: rRepo, cRepo: cRepo, cbSvc: cbSvc, cSvc: cSvc, peSvc: peSvc}
}

func (uc *buildCircuitUseCaseImpl) Handle(in BuildCircuitInput) (BuildCircuitOutput, error) {
	// hops 引数を service に渡して Circuit を生成
	exitID := vo.RelayID{}
	if in.ExitRelayID != "" {
		var err error
		exitID, err = vo.NewRelayID(in.ExitRelayID)
		if err != nil {
			return BuildCircuitOutput{}, err
		}
	}
	cir, err := uc.build(in.Hops, exitID)
	if err != nil {
		return BuildCircuitOutput{}, err
	}

	// 5. 保存
	if err := uc.cRepo.Save(cir); err != nil {
		_ = uc.cbSvc.TeardownCircuit(cir.Conn(0), cir.ID())
		cir.Conn(0).Close()
		return BuildCircuitOutput{}, fmt.Errorf("save circuit: %w", err)
	}

	out := BuildCircuitOutput{
		CircuitID: cir.ID(),
	}

	return out, nil
}

func (uc *buildCircuitUseCaseImpl) build(hops int, exit vo.RelayID) (cir *entity.Circuit, err error) {
	if hops <= 0 {
		hops = 3
	}
	// 1. オンラインリスト取得
	relays, err := uc.rRepo.AllOnline()
	if err != nil {
		return nil, fmt.Errorf("list relays: %w", err)
	}
	if len(relays) < hops {
		return nil, fmt.Errorf("not enough online relays (need %d)", hops)
	}

	var exitRelay *entity.Relay
	if exit != (vo.RelayID{}) {
		r, err := uc.rRepo.FindByID(exit)
		if err != nil {
			return nil, fmt.Errorf("exit relay not found: %w", err)
		}
		if r.Status() != entity.Online {
			return nil, fmt.Errorf("exit relay not online")
		}
		for i, rel := range relays {
			if rel.ID().Equal(exit) {
				exitRelay = rel
				relays = append(relays[:i], relays[i+1:]...)
				break
			}
		}
		if exitRelay == nil {
			// exit relay was not in online list
			return nil, fmt.Errorf("exit relay not in online list")
		}
		if hops == 1 {
			relays = []*entity.Relay{}
		}
	}

	// 2. 無作為に hops 個選出（重複なし）
	if err := secureShuffle(relays); err != nil {
		return nil, fmt.Errorf("shuffle relays: %w", err)
	}
	var selected []*entity.Relay
	if exitRelay == nil {
		selected = relays[:hops]
	} else {
		if hops-1 > len(relays) {
			return nil, fmt.Errorf("not enough online relays (need %d)", hops)
		}
		selected = append(selected, relays[:hops-1]...)
		selected = append(selected, exitRelay)
	}

	relayIDs := make([]vo.RelayID, 0, hops)
	keys := make([]vo.AESKey, hops)
	nonces := make([]vo.Nonce, hops)

	for _, r := range selected {
		relayIDs = append(relayIDs, r.ID())
	}

	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}
	priv := vo.NewRSAPrivKey(rawKey)

	// 3. CircuitID 生成
	cid := vo.NewCircuitID()

	// --- build circuit over the network ---
	const ioTimeout = 10 * time.Second
	dialCtx, cancel := context.WithTimeout(context.Background(), ioTimeout)
	defer cancel()
	type dialRes struct {
		c   net.Conn
		err error
	}
	// 4. 各 hop に ExtendCell を送信
	dch := make(chan dialRes, 1)
	go func() {
		c, err := uc.cbSvc.ConnectToRelay(selected[0].Endpoint().String())
		dch <- dialRes{c: c, err: err}
	}()

	// 待機して接続を取得
	var conn net.Conn
	select {
	case <-dialCtx.Done():
		return nil, fmt.Errorf("dial: %w", dialCtx.Err())
	case res := <-dch:
		if res.err != nil {
			return nil, res.err
		}
		conn = res.c
	}

	// 4. 各 hop に ExtendCell を送信
	for i := range selected {
		defer func() {
			if err != nil {
				_ = uc.cbSvc.TeardownCircuit(conn, cid)
				conn.Close()
			}
		}()
		// 4.1. 各 hop の鍵と nonce を生成
		cliPriv, cliPub, err := uc.cSvc.X25519Generate()
		if err != nil {
			return nil, fmt.Errorf("generate x25519 key: %w", err)
		}
		var pubArr [32]byte
		copy(pubArr[:], cliPub)

		nextHop := ""
		if i+1 < len(selected) {
			nextHop = selected[i+1].Endpoint().String()
		}

		// 4.2. ExtendCell を生成
		payload, err := uc.peSvc.EncodeExtendPayload(&service.ExtendPayloadDTO{
			NextHop:   nextHop,
			ClientPub: pubArr,
		})
		if err != nil {
			return nil, err
		}

		// 4.3. ExtendCell を送信
		cell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, vo.NewStreamIDControl(), payload)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(ioTimeout))
		if err := uc.cbSvc.SendExtendCell(conn, cell); err != nil {
			return nil, err
		}
		resp, err := uc.cbSvc.WaitForCreatedResponse(conn)
		if err != nil {
			return nil, err
		}

		// 4.4. レスポンスから鍵と nonce を生成
		conn.SetDeadline(time.Time{})
		created, err := uc.peSvc.DecodeCreatedPayload(resp)
		if err != nil {
			return nil, err
		}
		secret, err := uc.cSvc.X25519Shared(cliPriv, created.RelayPub[:])
		if err != nil {
			return nil, err
		}
		key, nonce, err := uc.cSvc.DeriveKeyNonce(secret)
		if err != nil {
			return nil, err
		}
		keys[i] = key
		nonces[i] = nonce
	}

	circuit, err := entity.NewCircuit(cid, relayIDs, keys, nonces, priv)
	if err != nil {
		return nil, err
	}

	// 5. 接続をセット
	circuit.SetConn(0, conn)

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
