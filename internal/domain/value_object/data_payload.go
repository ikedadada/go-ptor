package value_object


// DataPayload represents application data flowing through a circuit.
type DataPayload struct {
	StreamID uint16
	Data     []byte
}

// EncodeDataPayload encodes the payload using gob.
func EncodeDataPayload(p *DataPayload) ([]byte, error) {
	return EncodePayload(p)
}

// DecodeDataPayload decodes bytes into a DataPayload.
func DecodeDataPayload(b []byte) (*DataPayload, error) {
	return DecodePayload[DataPayload](b)
}
