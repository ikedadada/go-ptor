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
	csRepo repository.ConnStateRepository
	csSvc  service.CellSenderService
}

// NewHandleDestroyUseCase creates a new destroy use case
func NewHandleDestroyUseCase(csRepo repository.ConnStateRepository, csSvc service.CellSenderService) HandleDestroyUseCase {
	return &handleDestroyUseCaseImpl{
		csRepo: csRepo,
		csSvc:  csSvc,
	}
}

func (uc *handleDestroyUseCaseImpl) Destroy(st *entity.ConnState, cid vo.CircuitID) error {
	if st.Down() != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.csSvc.ForwardCell(st.Down(), cid, c)
	}
	_ = uc.csRepo.Delete(cid)
	return nil
}
