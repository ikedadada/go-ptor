package value_object

import (
	"ikedadada/go-ptor/internal/infrastructure/util"
)

// ExtendPayload carries the information needed to extend a circuit to the next hop.
// ExtendPayload carries the next hop address and the client's public key.
type ExtendPayload struct {
	NextHop   string
	ClientPub [32]byte
}

// EncodeExtendPayload serializes p using gob.
func EncodeExtendPayload(p *ExtendPayload) ([]byte, error) {
	return util.EncodePayload(p)
}

// DecodeExtendPayload decodes the payload from gob bytes.
func DecodeExtendPayload(b []byte) (*ExtendPayload, error) {
	return util.DecodePayload[ExtendPayload](b)
}
