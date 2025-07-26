package usecase

import (
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// DestroyUseCase handles circuit destruction operations
type DestroyUseCase interface {
	// Destroy destroys a circuit
	Destroy(st *entity.ConnState, cid vo.CircuitID) error
}

type destroyUseCaseImpl struct {
	repo   repository.ConnStateRepository
	sender service.CellSenderService
}

// NewDestroyUseCase creates a new destroy use case
func NewDestroyUseCase(repo repository.ConnStateRepository, sender service.CellSenderService) DestroyUseCase {
	return &destroyUseCaseImpl{
		repo:   repo,
		sender: sender,
	}
}

func (uc *destroyUseCaseImpl) Destroy(st *entity.ConnState, cid vo.CircuitID) error {
	if st.Down() != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.sender.ForwardCell(st.Down(), cid, c)
	}
	_ = uc.repo.Delete(cid)
	return nil
}
