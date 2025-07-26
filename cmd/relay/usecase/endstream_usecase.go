package usecase

import (
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// EndStreamUseCase handles stream termination operations
type EndStreamUseCase interface {
	// EndStream terminates a stream
	EndStream(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error
}

type endStreamUseCaseImpl struct {
	repo   repository.ConnStateRepository
	sender service.CellSenderService
}

// NewEndStreamUseCase creates a new end stream use case
func NewEndStreamUseCase(repo repository.ConnStateRepository, sender service.CellSenderService) EndStreamUseCase {
	return &endStreamUseCaseImpl{
		repo:   repo,
		sender: sender,
	}
}

func (uc *endStreamUseCaseImpl) EndStream(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error {
	var p *vo.DataPayload
	var err error
	if len(cell.Payload) > 0 {
		p, err = vo.DecodeDataPayload(cell.Payload)
		if err != nil {
			return err
		}
	} else {
		p = &vo.DataPayload{}
	}
	if p.StreamID == 0 {
		uc.repo.DestroyAllStreams(cid)
		if st.Down() != nil {
			ensureServeDown(st)
			_ = uc.sender.ForwardCell(st.Down(), cid, cell)
		}
		_ = uc.repo.Delete(cid)
		return nil
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	_ = uc.repo.RemoveStream(cid, sid)
	if st.Down() != nil {
		ensureServeDown(st)
		return uc.sender.ForwardCell(st.Down(), cid, cell)
	}
	return nil
}
