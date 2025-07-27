package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// HandleEndInput represents a received END cell.
type HandleEndInput struct {
	CircuitID string
	StreamID  uint16 // 0 means control END
}

// HandleEndOutput reports the result of closing streams.
type HandleEndOutput struct {
	Closed bool `json:"closed"`
}

// HandleEndUseCase processes incoming END cells.
type HandleEndUseCase interface {
	Handle(in HandleEndInput) (HandleEndOutput, error)
}

type handleEndUseCaseImpl struct {
	cRepo repository.CircuitRepository
}

// NewHandleEndUseCase creates a use case for handling END cells.
func NewHandleEndUseCase(cRepo repository.CircuitRepository) HandleEndUseCase {
	return &handleEndUseCaseImpl{cRepo: cRepo}
}

func (uc *handleEndUseCaseImpl) Handle(in HandleEndInput) (HandleEndOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return HandleEndOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}

	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return HandleEndOutput{}, fmt.Errorf("circuit not found: %w", err)
	}

	if in.StreamID == 0 {
		// close entire circuit
		for _, sid := range cir.ActiveStreams() {
			cir.CloseStream(sid)
		}
		_ = uc.cRepo.Delete(cid)
		return HandleEndOutput{Closed: true}, nil
	}

	sid, err := vo.StreamIDFrom(in.StreamID)
	if err != nil {
		return HandleEndOutput{}, fmt.Errorf("parse stream id: %w", err)
	}
	cir.CloseStream(sid)
	return HandleEndOutput{Closed: true}, nil
}
