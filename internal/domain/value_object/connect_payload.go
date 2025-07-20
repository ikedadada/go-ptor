package value_object


// ConnectPayload specifies the hidden service address for CONNECT command.
type ConnectPayload struct {
	Target string
}

// EncodeConnectPayload serializes p using gob.
func EncodeConnectPayload(p *ConnectPayload) ([]byte, error) {
	return EncodePayload(p)
}

// DecodeConnectPayload decodes bytes into a ConnectPayload.
func DecodeConnectPayload(b []byte) (*ConnectPayload, error) {
	return DecodePayload[ConnectPayload](b)
}
