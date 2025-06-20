package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/application/service"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type ShutdownCircuitInput struct {
	CircuitID string
}

type ShutdownCircuitOutput struct {
	Success bool `json:"success"`
}

type ShutdownCircuitUseCase interface {
	Handle(in ShutdownCircuitInput) (ShutdownCircuitOutput, error)
}

type shutdownCircuitUseCaseImpl struct {
	repo repository.CircuitRepository
	tx   service.CircuitTransmitter
}

func NewShutdownCircuitInteractor(cr repository.CircuitRepository, tx service.CircuitTransmitter) ShutdownCircuitUseCase {
	return &shutdownCircuitUseCaseImpl{repo: cr, tx: tx}
}

// Handle は回路内の全ストリームを END セルで閉じ、制御 END を送ってから
// CircuitRepository から削除する。
func (uc *shutdownCircuitUseCaseImpl) Handle(in ShutdownCircuitInput) (ShutdownCircuitOutput, error) {
	// --- 1. CircuitID パース & Circuit 取得
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	cir, err := uc.repo.Find(cid)
	if err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("circuit not found: %w", err)
	}

	// --- 2. アクティブストリームを順に閉じる
	for _, sid := range cir.ActiveStreams() {
		// ドメイン側を先に更新
		cir.CloseStream(sid)
		// ネットワーク側へ END セル
		_ = uc.tx.SendEnd(cid, sid) // 送信エラーは無視 or 集約
	}

	// --- 3. 制御ストリーム 0 で回路破棄を通知
	_ = uc.tx.SendEnd(cid, 0) // StreamID 0 は回路制御専用とする

	// --- 4. Repository から削除
	if err := uc.repo.Delete(cid); err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("repo delete: %w", err)
	}

	return ShutdownCircuitOutput{Success: true}, nil
}
