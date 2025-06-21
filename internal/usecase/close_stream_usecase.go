package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

type CloseStreamInput struct {
	CircuitID string
	StreamID  uint16
}

type CloseStreamOutput struct {
	Closed bool `json:"closed"`
}

type CloseStreamUseCase interface {
	Handle(in CloseStreamInput) (CloseStreamOutput, error)
}

type closeStreamUsecaseImpl struct {
	cr repository.CircuitRepository
	tx service.CircuitTransmitter
}

func NewCloseStreamUsecase(cr repository.CircuitRepository, tx service.CircuitTransmitter) CloseStreamUseCase {
	return &closeStreamUsecaseImpl{cr: cr, tx: tx}
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
	cir.CloseStream(sid)                            // ドメイン側の状態更新
	if err := uc.tx.SendEnd(cid, sid); err != nil { // END セル送信
		return CloseStreamOutput{}, err
	}
	return CloseStreamOutput{Closed: true}, nil
}
