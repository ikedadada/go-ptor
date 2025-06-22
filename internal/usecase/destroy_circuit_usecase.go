package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// DestroyCircuitInput triggers a DESTROY cell transmission.
type DestroyCircuitInput struct {
	CircuitID string
}

// DestroyCircuitOutput is returned after a circuit is aborted.
type DestroyCircuitOutput struct {
	Aborted bool `json:"aborted"`
}

// DestroyCircuitUseCase aborts an existing circuit.
type DestroyCircuitUseCase interface {
	Handle(in DestroyCircuitInput) (DestroyCircuitOutput, error)
}

type destroyCircuitUsecaseImpl struct {
	repo repository.CircuitRepository
	tx   service.CircuitTransmitter
}

// NewDestroyCircuitUsecase returns a use case to abort circuits.
func NewDestroyCircuitUsecase(r repository.CircuitRepository, tx service.CircuitTransmitter) DestroyCircuitUseCase {
	return &destroyCircuitUsecaseImpl{repo: r, tx: tx}
}

func (uc *destroyCircuitUsecaseImpl) Handle(in DestroyCircuitInput) (DestroyCircuitOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	// notify all relays about circuit tear-down
	cir, err := uc.repo.Find(cid)
	if err == nil && cir != nil {
		// send END for each active stream to allow graceful close
		for _, sid := range cir.ActiveStreams() {
			_ = uc.tx.SendEnd(cid, sid)
		}
	}
	if err := uc.tx.SendDestroy(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("send destroy: %w", err)
	}
	if err := uc.repo.Delete(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("repo delete: %w", err)
	}
	return DestroyCircuitOutput{Aborted: true}, nil
}
