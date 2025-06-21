package service

import (
	"crypto/rand"
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
	rr repository.RelayRepository
	cr repository.CircuitRepository
}

func NewCircuitBuildService(rr repository.RelayRepository, cr repository.CircuitRepository) CircuitBuildService {

	return &circuitBuildServiceImpl{rr: rr, cr: cr}
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
	keys := make([]value_object.AESKey, 0, hops)
	nonces := make([]value_object.Nonce, 0, hops)

	for _, r := range selected {
		relayIDs = append(relayIDs, r.ID())

		k, err := value_object.NewAESKey() // 32B ランダム
		if err != nil {
			return nil, fmt.Errorf("generate AES key: %w", err)
		}
		keys = append(keys, k)

		n, err := value_object.NewNonce() // 12B ランダム
		if err != nil {
			return nil, fmt.Errorf("generate nonce: %w", err)
		}
		nonces = append(nonces, n)
	}

	// 3. CircuitID 生成
	cid := value_object.NewCircuitID()

	// 4. Circuit エンティティ生成
	circuit, err := entity.NewCircuit(cid, relayIDs, keys, nonces)
	if err != nil {
		return nil, err
	}

	// 5. 保存
	if err := b.cr.Save(circuit); err != nil {
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
