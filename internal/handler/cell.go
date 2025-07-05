package handler

import (
	"io"

	"github.com/google/uuid"

	"ikedadada/go-ptor/internal/domain/value_object"
)

// ReadCell reads a circuit ID followed by a value_object.Cell from r.
// The payload is returned as-is and may still be encrypted.
func ReadCell(r io.Reader) (value_object.CircuitID, *value_object.Cell, error) {
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
	var cellBuf [value_object.MaxCellSize]byte
	if _, err := io.ReadFull(r, cellBuf[:]); err != nil {
		return value_object.CircuitID{}, nil, err
	}
	cell, err := value_object.Decode(cellBuf[:])
	if err != nil {
		return value_object.CircuitID{}, nil, err
	}
	return cid, cell, nil
}
