package value_object

import (
	"bytes"
	"encoding/gob"
)

// DataPayload represents application data flowing through a circuit.
type DataPayload struct {
	StreamID uint16
	Data     []byte
}

// EncodeDataPayload encodes the payload using gob.
func EncodeDataPayload(p *DataPayload) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(p)
	return buf.Bytes(), err
}

// DecodeDataPayload decodes bytes into a DataPayload.
func DecodeDataPayload(b []byte) (*DataPayload, error) {
	var p DataPayload
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&p)
	return &p, err
}
