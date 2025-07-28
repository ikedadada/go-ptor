package usecase

import (
	"fmt"
	"io"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// ReceiveCellInput specifies the circuit to receive data from
type ReceiveCellInput struct {
	CircuitID string
}

// ReceiveCellOutput contains the received cell data
type ReceiveCellOutput struct {
	Cell    *entity.Cell
	Circuit *entity.Circuit
	IsEOF   bool
}

// ReceiveCellUseCase handles receiving cells from circuit connections
type ReceiveCellUseCase interface {
	Handle(in ReceiveCellInput) (ReceiveCellOutput, error)
}

type receiveCellUseCaseImpl struct {
	cRepo repository.CircuitRepository
	crSvc service.CellReaderService
}

// NewReceiveCellUseCase creates a new use case for receiving cells
func NewReceiveCellUseCase(
	cRepo repository.CircuitRepository,
	crSvc service.CellReaderService,
) ReceiveCellUseCase {
	return &receiveCellUseCaseImpl{
		cRepo: cRepo,
		crSvc: crSvc,
	}
}

func (uc *receiveCellUseCaseImpl) Handle(in ReceiveCellInput) (ReceiveCellOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return ReceiveCellOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}

	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return ReceiveCellOutput{}, fmt.Errorf("find circuit: %w", err)
	}

	conn := cir.Conn(0)
	if conn == nil {
		return ReceiveCellOutput{}, fmt.Errorf("no connection for circuit")
	}

	// Read next cell from connection
	_, cell, err := uc.crSvc.ReadCell(conn)
	if err != nil {
		if err == io.EOF {
			return ReceiveCellOutput{IsEOF: true}, nil
		}
		return ReceiveCellOutput{}, fmt.Errorf("read cell: %w", err)
	}

	return ReceiveCellOutput{
		Cell:    cell,
		Circuit: cir,
		IsEOF:   false,
	}, nil
}
