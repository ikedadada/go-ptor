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

func NewTCPTransmitter(addr string) (service.CircuitTransmitter, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TCPTransmitter{conn: c}, nil
}

func (t *TCPTransmitter) send(cmd byte, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cell := value_object.Cell{Cmd: cmd, Version: value_object.Version, Payload: payload}
	buf, err := value_object.Encode(cell)
	if err != nil {
		return err
	}
	_, err = t.conn.Write(buf)
	return err
}

func (t *TCPTransmitter) SendData(_ value_object.CircuitID, s value_object.StreamID, d []byte) error {
	if len(d) > value_object.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16(), Data: d})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdData, p)
}

func (t *TCPTransmitter) SendEnd(_ value_object.CircuitID, s value_object.StreamID) error {
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16()})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdEnd, p)
}

func (t *TCPTransmitter) SendDestroy(value_object.CircuitID) error {
	return t.send(value_object.CmdDestroy, nil)
}
