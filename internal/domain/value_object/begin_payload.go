package value_object

import (
	"ikedadada/go-ptor/internal/infrastructure/util"
)

// BeginPayload specifies the target address for a new stream.
type BeginPayload struct {
	StreamID uint16
	Target   string
}

// EncodeBeginPayload encodes p using gob.
func EncodeBeginPayload(p *BeginPayload) ([]byte, error) {
	return util.EncodePayload(p)
}

// DecodeBeginPayload decodes bytes into a BeginPayload.
func DecodeBeginPayload(b []byte) (*BeginPayload, error) {
	return util.DecodePayload[BeginPayload](b)
}
