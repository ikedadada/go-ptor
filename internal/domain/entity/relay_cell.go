package entity

import "ikedadada/go-ptor/internal/domain/value_object"

// RelayCell represents a single relay cell exchanged between nodes.
// This is different from value_object.Cell which represents the low-level protocol cell.
type RelayCell struct {
	CircID   value_object.CircuitID
	StreamID value_object.StreamID
	Data     []byte
	End      bool
}
