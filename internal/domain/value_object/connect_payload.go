package value_object

import (
	"bytes"
	"encoding/gob"
)

// ConnectPayload specifies the hidden service address for CONNECT command.
type ConnectPayload struct {
	Target string
}

// EncodeConnectPayload serializes p using gob.
func EncodeConnectPayload(p *ConnectPayload) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(p)
	return buf.Bytes(), err
}

// DecodeConnectPayload decodes bytes into a ConnectPayload.
func DecodeConnectPayload(b []byte) (*ConnectPayload, error) {
	var p ConnectPayload
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&p)
	return &p, err
}
