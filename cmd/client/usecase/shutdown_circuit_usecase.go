package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// ShutdownCircuitInput specifies which circuit to close gracefully.
type ShutdownCircuitInput struct {
	CircuitID string
}

// ShutdownCircuitOutput reports whether shutdown succeeded.
type ShutdownCircuitOutput struct {
	Success bool `json:"success"`
}

// ShutdownCircuitUseCase closes all streams and removes the circuit.
type ShutdownCircuitUseCase interface {
	Handle(in ShutdownCircuitInput) (ShutdownCircuitOutput, error)
}

type shutdownCircuitUseCaseImpl struct {
	repo    repository.CircuitRepository
	factory service.MessagingServiceFactory
}

// NewShutdownCircuitUsecase returns a use case for orderly circuit shutdown.
func NewShutdownCircuitUsecase(cr repository.CircuitRepository, f service.MessagingServiceFactory) ShutdownCircuitUseCase {
	return &shutdownCircuitUseCaseImpl{repo: cr, factory: f}
}

// Handle は回路内の全ストリームを END セルで閉じ、制御 END を送ってから
// CircuitRepository から削除する。
func (uc *shutdownCircuitUseCaseImpl) Handle(in ShutdownCircuitInput) (ShutdownCircuitOutput, error) {
	// --- 1. CircuitID パース & Circuit 取得
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	cir, err := uc.repo.Find(cid)
	if err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	tx := uc.factory.New(cir.Conn(0))

	// --- 2. アクティブストリームを順に閉じる
	for _, sid := range cir.ActiveStreams() {
		// ドメイン側を先に更新
		cir.CloseStream(sid)
		// ネットワーク側へ END セル
		_ = tx.TerminateStream(cid, sid) // 送信エラーは無視 or 集約
	}

	// --- 3. 制御ストリーム 0 で回路破棄を通知
	_ = tx.TerminateStream(cid, 0) // StreamID 0 は回路制御専用とする

	// --- 4. Repository から削除
	if err := uc.repo.Delete(cid); err != nil {
		return ShutdownCircuitOutput{}, fmt.Errorf("repo delete: %w", err)
	}

	return ShutdownCircuitOutput{Success: true}, nil
}
