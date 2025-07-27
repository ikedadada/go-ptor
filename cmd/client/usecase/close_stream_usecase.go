package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/entity"
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

type closeStreamUseCaseImpl struct {
	cRepo   repository.CircuitRepository
	payload service.PayloadEncodingService
}

// NewCloseStreamUseCase creates a use case for closing streams.
func NewCloseStreamUseCase(cRepo repository.CircuitRepository, p service.PayloadEncodingService) CloseStreamUseCase {
	return &closeStreamUseCaseImpl{cRepo: cRepo, payload: p}
}

func (uc *closeStreamUseCaseImpl) Handle(in CloseStreamInput) (CloseStreamOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return CloseStreamOutput{}, err
	}
	sid, err := vo.StreamIDFrom(in.StreamID)
	if err != nil {
		return CloseStreamOutput{}, err
	}

	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return CloseStreamOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	cir.CloseStream(sid) // ドメイン側の状態更新

	// END セル送信
	payload, err := uc.payload.EncodeDataPayload(&service.DataPayloadDTO{StreamID: sid.UInt16()})
	if err != nil {
		return CloseStreamOutput{}, err
	}
	cell, err := entity.NewCell(vo.CmdEnd, payload)
	if err != nil {
		return CloseStreamOutput{}, err
	}
	if err := cell.SendToConnection(cir.Conn(0), cid); err != nil {
		return CloseStreamOutput{}, err
	}

	if len(cir.ActiveStreams()) == 0 { // 最後のストリームなら制御 END
		payload, err := uc.payload.EncodeDataPayload(&service.DataPayloadDTO{StreamID: 0})
		if err != nil {
			return CloseStreamOutput{}, err
		}
		cell, err := entity.NewCell(vo.CmdEnd, payload)
		if err != nil {
			return CloseStreamOutput{}, err
		}
		if err := cell.SendToConnection(cir.Conn(0), cid); err != nil {
			return CloseStreamOutput{}, err
		}
	}
	return CloseStreamOutput{Closed: true}, nil
}
