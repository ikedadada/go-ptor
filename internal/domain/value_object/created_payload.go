package value_object

import (
	"ikedadada/go-ptor/internal/infrastructure/util"
)

// CreatedPayload carries the relay's public key for a new circuit hop.
type CreatedPayload struct {
	RelayPub [32]byte
}

// EncodeCreatedPayload serializes p using gob.
func EncodeCreatedPayload(p *CreatedPayload) ([]byte, error) {
	return util.EncodePayload(p)
}

// DecodeCreatedPayload decodes from gob bytes.
func DecodeCreatedPayload(b []byte) (*CreatedPayload, error) {
	return util.DecodePayload[CreatedPayload](b)
}
