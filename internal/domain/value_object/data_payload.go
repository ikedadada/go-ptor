package value_object

import (
	"ikedadada/go-ptor/internal/infrastructure/util"
)

// DataPayload represents application data flowing through a circuit.
type DataPayload struct {
	StreamID uint16
	Data     []byte
}

// EncodeDataPayload encodes the payload using gob.
func EncodeDataPayload(p *DataPayload) ([]byte, error) {
	return util.EncodePayload(p)
}

// DecodeDataPayload decodes bytes into a DataPayload.
func DecodeDataPayload(b []byte) (*DataPayload, error) {
	return util.DecodePayload[DataPayload](b)
}
