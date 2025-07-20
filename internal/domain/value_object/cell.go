package value_object

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const (
	// Cell command types
	CmdExtend   CellCommand = 0x01
	CmdConnect  CellCommand = 0x02
	CmdData     CellCommand = 0x03
	CmdEnd      CellCommand = 0x04
	CmdDestroy  CellCommand = 0x05
	CmdBegin    CellCommand = 0x06
	CmdBeginAck CellCommand = 0x07
	CmdCreated  CellCommand = 0x08
)

// CellCommand represents the type of cell command in the Tor protocol
type CellCommand byte

// String returns the string representation of the cell command
func (c CellCommand) String() string {
	switch c {
	case CmdExtend:
		return "EXTEND"
	case CmdConnect:
		return "CONNECT"
	case CmdData:
		return "DATA"
	case CmdEnd:
		return "END"
	case CmdDestroy:
		return "DESTROY"
	case CmdBegin:
		return "BEGIN"
	case CmdBeginAck:
		return "BEGIN_ACK"
	case CmdCreated:
		return "CREATED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", byte(c))
	}
}

// IsValid checks if the command is a valid cell command
func (c CellCommand) IsValid() bool {
	switch c {
	case CmdExtend, CmdConnect, CmdData, CmdEnd, CmdDestroy, CmdBegin, CmdBeginAck, CmdCreated:
		return true
	default:
		return false
	}
}

const (
	// Protocol version constants
	ProtocolV1 ProtocolVersion = 0x01
)

// ProtocolVersion represents the protocol version used in cell communication
type ProtocolVersion byte

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

const (
	// Cell size constants
	MaxCellSize    = 512
	headerOverhead = 4 // CMD(1)+VER(1)+LEN(2)
	MaxPayloadSize = MaxCellSize - headerOverhead
)

// Cell represents a low-level 512-byte protocol cell used in Tor communication.
// This is the fundamental unit of communication between nodes in the network.
// For higher-level relay operations, see entity.RelayCell.
type Cell struct {
	Cmd     CellCommand     // Command type (CmdExtend, CmdData, etc.)
	Version ProtocolVersion // Protocol version
	Payload []byte          // Cell payload data
}

// Encode serializes the cell into a fixed 512-byte slice with random padding.
// Format: [CMD(1)] [VER(1)] [LEN(2)] [PAYLOAD(LEN)] [PADDING...]
func Encode(c Cell) ([]byte, error) {
	if len(c.Payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload too big: %d > %d", len(c.Payload), MaxPayloadSize)
	}
	buf := make([]byte, MaxCellSize)
	buf[0] = byte(c.Cmd)
	buf[1] = byte(c.Version)
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(c.Payload)))
	copy(buf[4:], c.Payload)
	if _, err := rand.Read(buf[4+len(c.Payload):]); err != nil {
		return nil, err
	}
	return buf, nil
}

// Decode parses a 512-byte buffer into a Cell struct.
// Format: [CMD(1)] [VER(1)] [LEN(2)] [PAYLOAD(LEN)] [PADDING...]
func Decode(buf []byte) (*Cell, error) {
	if len(buf) != MaxCellSize {
		return nil, fmt.Errorf("invalid cell length: %d", len(buf))
	}
	cmd := CellCommand(buf[0])
	if !cmd.IsValid() {
		return nil, fmt.Errorf("invalid cell command: %d", buf[0])
	}
	version := ProtocolVersion(buf[1])
	if !version.IsSupported() {
		return nil, fmt.Errorf("unsupported protocol version: %d", buf[1])
	}
	l := binary.BigEndian.Uint16(buf[2:4])
	if l > MaxPayloadSize {
		return nil, fmt.Errorf("invalid payload length: %d", l)
	}
	payload := make([]byte, l)
	copy(payload, buf[4:4+int(l)])
	return &Cell{
		Cmd:     cmd,
		Version: version,
		Payload: payload,
	}, nil
}
