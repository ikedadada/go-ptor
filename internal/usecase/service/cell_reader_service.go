package service

import (
	"io"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"

	"github.com/google/uuid"
)

// CellReader abstracts reading a circuit ID and cell from an io.Reader.
type CellReaderService interface {
	ReadCell(r io.Reader) (value_object.CircuitID, *entity.Cell, error)
}

// ProtocolCellReader implements CellReader using util.ReadCell for low-level protocol cells.
type cellReaderService struct{}

// NewProtocolCellReader returns a CellReader backed by util.ReadCell.
func NewCellReaderService() CellReaderService { return cellReaderService{} }

func (cellReaderService) ReadCell(r io.Reader) (value_object.CircuitID, *entity.Cell, error) {
	var idBuf [16]byte
	if _, err := io.ReadFull(r, idBuf[:]); err != nil {
		return value_object.CircuitID{}, nil, err
	}
	var id uuid.UUID
	copy(id[:], idBuf[:])
	cid, err := value_object.CircuitIDFrom(id.String())
	if err != nil {
		return value_object.CircuitID{}, nil, err
	}
	var cellBuf [entity.MaxCellSize]byte
	if _, err := io.ReadFull(r, cellBuf[:]); err != nil {
		return value_object.CircuitID{}, nil, err
	}
	cell, err := entity.Decode(cellBuf[:])
	if err != nil {
		return value_object.CircuitID{}, nil, err
	}
	return cid, cell, nil
}
