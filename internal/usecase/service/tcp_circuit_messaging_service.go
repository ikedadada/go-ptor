package service

import (
	"fmt"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitMessagingService はセル転送を担当するサービス。
// 回路 ID + ストリーム ID + データを受け取り、セル化してネットワークに送る。
type CircuitMessagingService interface {
	TransmitData(c value_object.CircuitID, s value_object.StreamID, data []byte) error
	InitiateStream(c value_object.CircuitID, s value_object.StreamID, data []byte) error
	EstablishConnection(c value_object.CircuitID, data []byte) error
	TerminateStream(c value_object.CircuitID, s value_object.StreamID) error
	DestroyCircuit(c value_object.CircuitID) error
}

// MessagingServiceFactory produces a CircuitMessagingService bound to a given connection.
type MessagingServiceFactory interface {
	New(conn net.Conn) CircuitMessagingService
}

// TCPMessagingServiceFactory creates TCPCircuitMessagingService instances.
type TCPMessagingServiceFactory struct{}

// New returns a CircuitMessagingService using the provided connection.
func (TCPMessagingServiceFactory) New(conn net.Conn) CircuitMessagingService {
	return NewTCPCircuitMessagingService(conn)
}

type TCPCircuitMessagingService struct {
	mu   sync.Mutex
	conn net.Conn
}

// NewTCPCircuitMessagingService wraps an already-connected net.Conn.
// The caller is responsible for establishing the connection.
func NewTCPCircuitMessagingService(conn net.Conn) CircuitMessagingService {
	return &TCPCircuitMessagingService{conn: conn}
}

func (t *TCPCircuitMessagingService) send(cmd value_object.CellCommand, cid value_object.CircuitID, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cell := entity.Cell{Cmd: cmd, Version: value_object.ProtocolV1, Payload: payload}
	buf, err := entity.Encode(cell)
	if err != nil {
		return err
	}
	packet := append(cid.Bytes(), buf...)
	_, err = t.conn.Write(packet)
	return err
}

func (t *TCPCircuitMessagingService) TransmitData(cid value_object.CircuitID, s value_object.StreamID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16(), Data: d})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdData, cid, p)
}

func (t *TCPCircuitMessagingService) InitiateStream(cid value_object.CircuitID, _ value_object.StreamID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(value_object.CmdBegin, cid, d)
}

func (t *TCPCircuitMessagingService) EstablishConnection(cid value_object.CircuitID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(value_object.CmdConnect, cid, d)
}

func (t *TCPCircuitMessagingService) TerminateStream(cid value_object.CircuitID, s value_object.StreamID) error {
	p, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: s.UInt16()})
	if err != nil {
		return err
	}
	return t.send(value_object.CmdEnd, cid, p)
}

func (t *TCPCircuitMessagingService) DestroyCircuit(cid value_object.CircuitID) error {
	return t.send(value_object.CmdDestroy, cid, nil)
}
