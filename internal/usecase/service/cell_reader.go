package service

import (
	"io"

	"ikedadada/go-ptor/internal/domain/value_object"
)

// CellReader abstracts reading a circuit ID and cell from an io.Reader.
type CellReader interface {
	ReadCell(r io.Reader) (value_object.CircuitID, *value_object.Cell, error)
}
