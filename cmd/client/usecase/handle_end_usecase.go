package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// HandleEndInput represents a received END cell.
type HandleEndInput struct {
	CircuitID vo.CircuitID
	StreamID  vo.StreamID // 0 means control END
}

// HandleEndUseCase processes incoming END cells.
type HandleEndUseCase interface {
	Handle(in HandleEndInput) error
}

type handleEndUseCaseImpl struct {
	cRepo repository.CircuitRepository
}

// NewHandleEndUseCase creates a use case for handling END cells.
func NewHandleEndUseCase(cRepo repository.CircuitRepository) HandleEndUseCase {
	return &handleEndUseCaseImpl{cRepo: cRepo}
}

func (uc *handleEndUseCaseImpl) Handle(in HandleEndInput) error {
	cid := in.CircuitID

	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return fmt.Errorf("circuit not found: %w", err)
	}

	if in.StreamID.Equal(0) {
		// close entire circuit
		for _, sid := range cir.ActiveStreams() {
			cir.CloseStream(sid)
		}
		_ = uc.cRepo.Delete(cid)
		return nil
	}

	// close specific stream
	cir.CloseStream(in.StreamID)
	return nil
}
