package usecase

import (
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleDestroyUseCase handles circuit destruction operations
type HandleDestroyUseCase interface {
	// Destroy destroys a circuit
	Destroy(st *entity.ConnState, cid vo.CircuitID) error
}

type handleDestroyUseCaseImpl struct {
	repo   repository.ConnStateRepository
	sender service.CellSenderService
}

// NewHandleDestroyUseCase creates a new destroy use case
func NewHandleDestroyUseCase(repo repository.ConnStateRepository, sender service.CellSenderService) HandleDestroyUseCase {
	return &handleDestroyUseCaseImpl{
		repo:   repo,
		sender: sender,
	}
}

func (uc *handleDestroyUseCaseImpl) Destroy(st *entity.ConnState, cid vo.CircuitID) error {
	if st.Down() != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.sender.ForwardCell(st.Down(), cid, c)
	}
	_ = uc.repo.Delete(cid)
	return nil
}
