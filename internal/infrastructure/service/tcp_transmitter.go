package service

import (
	"fmt"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service" // 依存関係のために必要
)

type TCPTransmitter struct {
	mu   sync.Mutex
	conn net.Conn
}

// NewTCPTransmitter wraps an already-connected net.Conn.
// The caller is responsible for establishing the connection.
func NewTCPTransmitter(conn net.Conn) service.CircuitTransmitter {
	return &TCPTransmitter{conn: conn}
}

func (t *TCPTransmitter) send(cmd value_object.CellCommand, cid value_object.CircuitID, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cell := value_object.Cell{Cmd: cmd, Version: value_object.ProtocolV1, Payload: payload}
	buf, err := value_object.Encode(cell)
	if err != nil {
		return err
	}
	packet := append(cid.Bytes(), buf...)
	_, err = t.conn.Write(packet)
	return err
}

func (t *TCPTransmitter) SendData(cid value_object.CircuitID, s value_object.StreamID, d []byte) error {
	if len(d) > value_object.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16(), Data: d})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdData, cid, p)
}

func (t *TCPTransmitter) SendBegin(cid value_object.CircuitID, _ value_object.StreamID, d []byte) error {
	if len(d) > value_object.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(value_object.CmdBegin, cid, d)
}

func (t *TCPTransmitter) SendConnect(cid value_object.CircuitID, d []byte) error {
	if len(d) > value_object.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(value_object.CmdConnect, cid, d)
}

func (t *TCPTransmitter) SendEnd(cid value_object.CircuitID, s value_object.StreamID) error {
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16()})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdEnd, cid, p)
}

func (t *TCPTransmitter) SendDestroy(cid value_object.CircuitID) error {
	return t.send(value_object.CmdDestroy, cid, nil)
}
