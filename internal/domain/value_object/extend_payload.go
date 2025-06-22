package value_object

import (
	"bytes"
	"encoding/gob"
)

// ExtendPayload carries the information needed to extend a circuit to the next hop.
// ExtendPayload carries the next hop address and the client's public key.
type ExtendPayload struct {
	NextHop   string
	ClientPub [32]byte
}

// EncodeExtendPayload serializes p using gob.
func EncodeExtendPayload(p *ExtendPayload) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(p)
	return buf.Bytes(), err
}

// DecodeExtendPayload decodes the payload from gob bytes.
func DecodeExtendPayload(b []byte) (*ExtendPayload, error) {
	var p ExtendPayload
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&p)
	return &p, err
}
