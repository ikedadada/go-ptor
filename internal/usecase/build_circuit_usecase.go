// file: internal/usecase/build_circuit_usecase.go
package usecase

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// ---------- DTO ----------

// BuildCircuitInput はユーザーが指定できるパラメータ
type BuildCircuitInput struct {
	Hops int // 省略時はデフォルト (3)
}

// BuildCircuitOutput は UI / API に返すレスポンス
type BuildCircuitOutput struct {
	CircuitID string                  `json:"circuit_id"`
	Hops      []string                `json:"relay_ids"`
	Keys      [][]byte                `json:"aes_keys"` // Base64 等はプレゼン層で
	Nonces    [][]byte                `json:"nonces"`   // 同上
	AddrList  []value_object.Endpoint `json:"endpoints"`
}

// ---------- UseCase インターフェース ----------

type BuildCircuitUseCase interface {
	Handle(input BuildCircuitInput) (BuildCircuitOutput, error)
}

// ---------- 実装 ----------

type buildCircuitUseCaseImpl struct {
	builder service.CircuitBuildService
}

// コンストラクタ
func NewBuildCircuitUseCase(b service.CircuitBuildService) BuildCircuitUseCase {
	return &buildCircuitUseCaseImpl{builder: b}
}

func (uc *buildCircuitUseCaseImpl) Handle(in BuildCircuitInput) (BuildCircuitOutput, error) {
	// hops 引数を service に渡して Circuit を生成
	cir, err := uc.builder.Build(in.Hops) // Build(hops int) を想定
	if err != nil {
		return BuildCircuitOutput{}, err
	}

	out := BuildCircuitOutput{
		CircuitID: cir.ID().String(),
	}

	// Relay ID
	for _, rid := range cir.Hops() {
		out.Hops = append(out.Hops, rid.String())
	}

	// 鍵・ノンス（array → slice 変換）
	for i := range cir.Hops() {
		key := cir.HopKey(i)     // [32]byte
		nonce := cir.HopNonce(i) // [12]byte
		out.Keys = append(out.Keys, key[:])
		out.Nonces = append(out.Nonces, nonce[:])
	}

	return out, nil
}
