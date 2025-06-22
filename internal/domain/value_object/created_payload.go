package value_object

import (
	"bytes"
	"encoding/gob"
)

// CreatedPayload carries the relay's public key for a new circuit hop.
type CreatedPayload struct {
	RelayPub [32]byte
}

// EncodeCreatedPayload serializes p using gob.
func EncodeCreatedPayload(p *CreatedPayload) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(p)
	return buf.Bytes(), err
}

// DecodeCreatedPayload decodes from gob bytes.
func DecodeCreatedPayload(b []byte) (*CreatedPayload, error) {
	var p CreatedPayload
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&p)
	return &p, err
}
