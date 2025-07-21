package service

import (
	"io"

	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"

	"github.com/google/uuid"
)

// CellReader abstracts reading a circuit ID and cell from an io.Reader.
type CellReaderService interface {
	ReadCell(r io.Reader) (vo.CircuitID, *entity.Cell, error)
}

// ProtocolCellReader implements CellReader using util.ReadCell for low-level protocol cells.
type cellReaderService struct{}

// NewProtocolCellReader returns a CellReader backed by util.ReadCell.
func NewCellReaderService() CellReaderService { return cellReaderService{} }

func (cellReaderService) ReadCell(r io.Reader) (vo.CircuitID, *entity.Cell, error) {
	var idBuf [16]byte
	if _, err := io.ReadFull(r, idBuf[:]); err != nil {
		return vo.CircuitID{}, nil, err
	}
	var id uuid.UUID
	copy(id[:], idBuf[:])
	cid, err := vo.CircuitIDFrom(id.String())
	if err != nil {
		return vo.CircuitID{}, nil, err
	}
	var cellBuf [entity.MaxCellSize]byte
	if _, err := io.ReadFull(r, cellBuf[:]); err != nil {
		return vo.CircuitID{}, nil, err
	}
	cell, err := entity.Decode(cellBuf[:])
	if err != nil {
		return vo.CircuitID{}, nil, err
	}
	return cid, cell, nil
}
