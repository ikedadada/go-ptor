package value_object

import "fmt"

// ProtocolVersion represents the protocol version used in cell communication
type ProtocolVersion byte

const (
	// Protocol version constants
	ProtocolV1 ProtocolVersion = 0x01
)

// String returns the string representation of the protocol version
func (v ProtocolVersion) String() string {
	switch v {
	case ProtocolV1:
		return "v1"
	default:
		return fmt.Sprintf("unknown(%d)", byte(v))
	}
}

// IsSupported checks if the protocol version is supported
func (v ProtocolVersion) IsSupported() bool {
	switch v {
	case ProtocolV1:
		return true
	default:
		return false
	}
}