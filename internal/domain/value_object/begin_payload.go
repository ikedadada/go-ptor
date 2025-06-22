package value_object

import (
	"bytes"
	"encoding/gob"
)

// BeginPayload specifies the target address for a new stream.
type BeginPayload struct {
	StreamID uint16
	Target   string
}

// EncodeBeginPayload encodes p using gob.
func EncodeBeginPayload(p *BeginPayload) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(p)
	return buf.Bytes(), err
}

// DecodeBeginPayload decodes bytes into a BeginPayload.
func DecodeBeginPayload(b []byte) (*BeginPayload, error) {
	var p BeginPayload
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&p)
	return &p, err
}
