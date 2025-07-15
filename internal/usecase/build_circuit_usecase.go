// file: internal/usecase/build_circuit_usecase.go
package usecase

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// ---------- DTO ----------

// BuildCircuitInput はユーザーが指定できるパラメータ
type BuildCircuitInput struct {
	Hops        int    // 省略時はデフォルト (3)
	ExitRelayID string // 任意。指定時は最終 hop をこのリレーに固定
}

// BuildCircuitOutput は UI / API に返すレスポンス
type BuildCircuitOutput struct {
	CircuitID string                  `json:"circuit_id"`
	Hops      []string                `json:"relay_ids"`
	Keys      [][]byte                `json:"aes_keys"` // Base64 等はプレゼン層で
	Nonces    [][]byte                `json:"nonces"`   // 同上
	AddrList  []value_object.Endpoint `json:"endpoints"`
}

// BuildCircuitUseCase creates new circuits according to the input parameters.
type BuildCircuitUseCase interface {
	Handle(input BuildCircuitInput) (BuildCircuitOutput, error)
}

// ---------- 実装 ----------

type buildCircuitUseCaseImpl struct {
	builder service.CircuitBuildService
}

// NewBuildCircuitUseCase creates a use case for building circuits.
func NewBuildCircuitUseCase(b service.CircuitBuildService) BuildCircuitUseCase {
	return &buildCircuitUseCaseImpl{builder: b}
}

func (uc *buildCircuitUseCaseImpl) Handle(in BuildCircuitInput) (BuildCircuitOutput, error) {
	// hops 引数を service に渡して Circuit を生成
	exitID := value_object.RelayID{}
	if in.ExitRelayID != "" {
		var err error
		exitID, err = value_object.NewRelayID(in.ExitRelayID)
		if err != nil {
			return BuildCircuitOutput{}, err
		}
	}
	cir, err := uc.builder.Build(in.Hops, exitID)
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
		nonce := cir.HopBaseNonce(i) // [12]byte - use base nonce for circuit info
		out.Keys = append(out.Keys, key[:])
		out.Nonces = append(out.Nonces, nonce[:])
	}

	return out, nil
}
