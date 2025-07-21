package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// CloseStreamInput identifies the stream to close on a circuit.
type CloseStreamInput struct {
	CircuitID string
	StreamID  uint16
}

// CloseStreamOutput indicates whether the close operation succeeded.
type CloseStreamOutput struct {
	Closed bool `json:"closed"`
}

// CloseStreamUseCase terminates an existing stream.
type CloseStreamUseCase interface {
	Handle(in CloseStreamInput) (CloseStreamOutput, error)
}

type closeStreamUsecaseImpl struct {
	cr      repository.CircuitRepository
	factory useSvc.MessagingServiceFactory
}

// NewCloseStreamUsecase creates a use case for closing streams.
func NewCloseStreamUsecase(cr repository.CircuitRepository, f useSvc.MessagingServiceFactory) CloseStreamUseCase {
	return &closeStreamUsecaseImpl{cr: cr, factory: f}
}

func (uc *closeStreamUsecaseImpl) Handle(in CloseStreamInput) (CloseStreamOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return CloseStreamOutput{}, err
	}
	sid, err := value_object.StreamIDFrom(in.StreamID)
	if err != nil {
		return CloseStreamOutput{}, err
	}

	cir, err := uc.cr.Find(cid)
	if err != nil {
		return CloseStreamOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	cir.CloseStream(sid) // ドメイン側の状態更新
	tx := uc.factory.New(cir.Conn(0))
	if err := tx.TerminateStream(cid, sid); err != nil { // END セル送信
		return CloseStreamOutput{}, err
	}
	if len(cir.ActiveStreams()) == 0 { // 最後のストリームなら制御 END
		if err := tx.TerminateStream(cid, 0); err != nil {
			return CloseStreamOutput{}, err
		}
	}
	return CloseStreamOutput{Closed: true}, nil
}
