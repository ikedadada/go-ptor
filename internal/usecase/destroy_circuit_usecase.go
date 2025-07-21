package usecase

import (
	"fmt"
	"net"

	"ikedadada/go-ptor/internal/domain/repository"
	vo "ikedadada/go-ptor/internal/domain/value_object"
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
	repo    repository.CircuitRepository
	factory service.MessagingServiceFactory
}

// NewDestroyCircuitUsecase returns a use case to abort circuits.
func NewDestroyCircuitUsecase(r repository.CircuitRepository, f service.MessagingServiceFactory) DestroyCircuitUseCase {
	return &destroyCircuitUsecaseImpl{repo: r, factory: f}
}

func (uc *destroyCircuitUsecaseImpl) Handle(in DestroyCircuitInput) (DestroyCircuitOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
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
			_ = tx.TerminateStream(cid, sid)
		}
	}
	if err := tx.DestroyCircuit(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("send destroy: %w", err)
	}
	if err := uc.repo.Delete(cid); err != nil {
		return DestroyCircuitOutput{}, fmt.Errorf("repo delete: %w", err)
	}
	return DestroyCircuitOutput{Aborted: true}, nil
}
