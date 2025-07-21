package value_object

// ExtendPayload carries the information needed to extend a circuit to the next hop.
// ExtendPayload carries the next hop address and the client's public key.
type ExtendPayload struct {
	NextHop   string
	ClientPub [32]byte
}

// EncodeExtendPayload serializes p using gob.
func EncodeExtendPayload(p *ExtendPayload) ([]byte, error) {
	return EncodePayload(p)
}

// DecodeExtendPayload decodes the payload from gob bytes.
func DecodeExtendPayload(b []byte) (*ExtendPayload, error) {
	return DecodePayload[ExtendPayload](b)
}
