package value_object

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const (
	Version byte = 0x01

	CmdExtend   byte = 0x01
	CmdConnect  byte = 0x02
	CmdData     byte = 0x03
	CmdEnd      byte = 0x04
	CmdDestroy  byte = 0x05
	CmdBegin    byte = 0x06
	CmdBeginAck byte = 0x07

	MaxPayloadSize = MaxCellSize - headerOverhead
)

// Cell represents a 512-byte protocol cell.
type Cell struct {
	Cmd     byte
	Version byte
	Payload []byte
}

// Encode serializes the cell into a fixed 512-byte slice with random padding.
func Encode(c Cell) ([]byte, error) {
	if len(c.Payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload too big: %d > %d", len(c.Payload), MaxPayloadSize)
	}
	buf := make([]byte, MaxCellSize)
	buf[0] = c.Cmd
	buf[1] = c.Version
	binary.BigEndian.PutUint16(buf[2:], uint16(len(c.Payload)))
	copy(buf[4:], c.Payload)
	if _, err := rand.Read(buf[4+len(c.Payload):]); err != nil {
		return nil, err
	}
	return buf, nil
}

// Decode parses a 512-byte buffer into a Cell struct.
func Decode(buf []byte) (*Cell, error) {
	if len(buf) != MaxCellSize {
		return nil, fmt.Errorf("invalid cell length: %d", len(buf))
	}
	l := binary.BigEndian.Uint16(buf[2:4])
	if l > MaxPayloadSize {
		return nil, fmt.Errorf("invalid payload length: %d", l)
	}
	payload := make([]byte, l)
	copy(payload, buf[4:4+int(l)])
	return &Cell{
		Cmd:     buf[0],
		Version: buf[1],
		Payload: payload,
	}, nil
}
