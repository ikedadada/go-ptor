package service

import (
	"io"

	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/handler"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// HandlerCellReader implements CellReader using handler.ReadCell.
type HandlerCellReader struct{}

// NewHandlerCellReader returns a CellReader backed by handler.ReadCell.
func NewHandlerCellReader() useSvc.CellReader { return HandlerCellReader{} }

func (HandlerCellReader) ReadCell(r io.Reader) (value_object.CircuitID, *value_object.Cell, error) {
	return handler.ReadCell(r)
}
