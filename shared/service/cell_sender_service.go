package service

import (
	"net"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// CellSenderService handles sending various types of cells over connections
type CellSenderService interface {
	// SendCreated sends a Created cell with the given payload
	SendCreated(w net.Conn, cid vo.CircuitID, payload []byte) error

	// SendAck sends a BeginAck cell
	SendAck(w net.Conn, cid vo.CircuitID) error

	// ForwardCell sends any cell with the circuit ID prepended
	ForwardCell(w net.Conn, cid vo.CircuitID, cell *entity.Cell) error
}
