package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
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
	factory service.MessagingServiceFactory
}

// NewCloseStreamUsecase creates a use case for closing streams.
func NewCloseStreamUsecase(cr repository.CircuitRepository, f service.MessagingServiceFactory) CloseStreamUseCase {
	return &closeStreamUsecaseImpl{cr: cr, factory: f}
}

func (uc *closeStreamUsecaseImpl) Handle(in CloseStreamInput) (CloseStreamOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return CloseStreamOutput{}, err
	}
	sid, err := vo.StreamIDFrom(in.StreamID)
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
