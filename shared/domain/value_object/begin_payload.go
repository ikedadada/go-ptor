package value_object

// BeginPayload specifies the target address for a new stream.
type BeginPayload struct {
	StreamID uint16
	Target   string
}

// EncodeBeginPayload encodes p using gob.
func EncodeBeginPayload(p *BeginPayload) ([]byte, error) {
	return EncodePayload(p)
}

// DecodeBeginPayload decodes bytes into a BeginPayload.
func DecodeBeginPayload(b []byte) (*BeginPayload, error) {
	return DecodePayload[BeginPayload](b)
}
