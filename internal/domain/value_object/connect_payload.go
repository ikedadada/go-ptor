package value_object

import (
	"ikedadada/go-ptor/internal/infrastructure/util"
)

// ConnectPayload specifies the hidden service address for CONNECT command.
type ConnectPayload struct {
	Target string
}

// EncodeConnectPayload serializes p using gob.
func EncodeConnectPayload(p *ConnectPayload) ([]byte, error) {
	return util.EncodePayload(p)
}

// DecodeConnectPayload decodes bytes into a ConnectPayload.
func DecodeConnectPayload(b []byte) (*ConnectPayload, error) {
	return util.DecodePayload[ConnectPayload](b)
}
