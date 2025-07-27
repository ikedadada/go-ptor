package usecase

import (
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleEndStreamUseCase handles stream termination operations
type HandleEndStreamUseCase interface {
	// EndStream terminates a stream
	EndStream(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error
}

type handleEndStreamUseCaseImpl struct {
	csRepo repository.ConnStateRepository
	csSvc  service.CellSenderService
	peSvc  service.PayloadEncodingService
}

// NewHandleEndStreamUseCase creates a new end stream use case
func NewHandleEndStreamUseCase(csRepo repository.ConnStateRepository, csSvc service.CellSenderService, peSvc service.PayloadEncodingService) HandleEndStreamUseCase {
	return &handleEndStreamUseCaseImpl{
		csRepo: csRepo,
		csSvc:  csSvc,
		peSvc:  peSvc,
	}
}

func (uc *handleEndStreamUseCaseImpl) EndStream(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error {
	var p *service.DataPayloadDTO
	var err error
	if len(cell.Payload) > 0 {
		p, err = uc.peSvc.DecodeDataPayload(cell.Payload)
		if err != nil {
			return err
		}
	} else {
		p = &service.DataPayloadDTO{}
	}
	if p.StreamID == 0 {
		uc.csRepo.DestroyAllStreams(cid)
		if st.Down() != nil {
			ensureServeDown(st)
			_ = uc.csSvc.ForwardCell(st.Down(), cid, cell)
		}
		_ = uc.csRepo.Delete(cid)
		return nil
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	_ = uc.csRepo.RemoveStream(cid, sid)
	if st.Down() != nil {
		ensureServeDown(st)
		return uc.csSvc.ForwardCell(st.Down(), cid, cell)
	}
	return nil
}
