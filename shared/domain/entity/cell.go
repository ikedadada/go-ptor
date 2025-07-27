package entity

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"net"
)

const (
	// Cell size constants
	MaxCellSize    = 512
	headerOverhead = 4 // CMD(1)+VER(1)+LEN(2)
	MaxPayloadSize = MaxCellSize - headerOverhead
)

// Cell represents a low-level 512-byte protocol cell used in Tor communication.
// This is the fundamental unit of communication between nodes in the network.
// For higher-level relay operations, see aggregate.RelayCell.
type Cell struct {
	Cmd     vo.CellCommand     // Command type (CmdExtend, CmdData, etc.)
	Version vo.ProtocolVersion // Protocol version
	Payload []byte             // Cell payload data
}

// NewCell creates a new Cell with the specified command and payload.
// The version defaults to ProtocolV1.
func NewCell(cmd vo.CellCommand, payload []byte) (*Cell, error) {
	if len(payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload too big: %d > %d", len(payload), MaxPayloadSize)
	}
	return &Cell{
		Cmd:     cmd,
		Version: vo.ProtocolV1,
		Payload: payload,
	}, nil
}

// SendToConnection encodes the cell and sends it over the connection with circuit ID prefix.
func (c *Cell) SendToConnection(conn net.Conn, cid vo.CircuitID) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}
	buf, err := Encode(*c)
	if err != nil {
		return err
	}
	packet := append(cid.Bytes(), buf...)
	_, err = conn.Write(packet)
	return err
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
	cmd := vo.CellCommand(buf[0])
	if !cmd.IsValid() {
		return nil, fmt.Errorf("invalid cell command: %d", buf[0])
	}
	version := vo.ProtocolVersion(buf[1])
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
