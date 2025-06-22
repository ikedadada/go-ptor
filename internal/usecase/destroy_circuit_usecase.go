package usecase

import (
	"fmt"
	"net"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
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
	repo    repository.CircuitRepository
	factory infraSvc.TransmitterFactory
}

// NewDestroyCircuitUsecase returns a use case to abort circuits.
func NewDestroyCircuitUsecase(r repository.CircuitRepository, f infraSvc.TransmitterFactory) DestroyCircuitUseCase {
	return &destroyCircuitUsecaseImpl{repo: r, factory: f}
}

func (uc *destroyCircuitUsecaseImpl) Handle(in DestroyCircuitInput) (DestroyCircuitOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	// notify all relays about circuit tear-down
	cir, err := uc.repo.Find(cid)
	var conn net.Conn
	if err == nil && cir != nil {
		conn = cir.Conn(0)
	}
	tx := uc.factory.New(conn)
	if err == nil && cir != nil {
		// send END for each active stream to allow graceful close
		for _, sid := range cir.ActiveStreams() {
			_ = tx.SendEnd(cid, sid)
		}
	}
	if err := tx.SendDestroy(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("send destroy: %w", err)
	}
	if err := uc.repo.Delete(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("repo delete: %w", err)
	}
	return DestroyCircuitOutput{Aborted: true}, nil
}
