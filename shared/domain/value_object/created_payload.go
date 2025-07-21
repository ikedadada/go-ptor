package value_object

// CreatedPayload carries the relay's public key for a new circuit hop.
type CreatedPayload struct {
	RelayPub [32]byte
}

// EncodeCreatedPayload serializes p using gob.
func EncodeCreatedPayload(p *CreatedPayload) ([]byte, error) {
	return EncodePayload(p)
}

// DecodeCreatedPayload decodes from gob bytes.
func DecodeCreatedPayload(b []byte) (*CreatedPayload, error) {
	return DecodePayload[CreatedPayload](b)
}
