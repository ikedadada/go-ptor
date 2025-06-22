package entity

import "ikedadada/go-ptor/internal/domain/value_object"

// Cell represents a single relay cell exchanged between nodes.
type Cell struct {
	CircID   value_object.CircuitID
	StreamID value_object.StreamID
	Data     []byte
	End      bool
}
