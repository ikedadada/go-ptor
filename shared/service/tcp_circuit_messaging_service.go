package service

import (
	"fmt"
	"net"
	"sync"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// CircuitMessagingService はセル転送を担当するサービス。
// 回路 ID + ストリーム ID + データを受け取り、セル化してネットワークに送る。
type CircuitMessagingService interface {
	TransmitData(c vo.CircuitID, s vo.StreamID, data []byte) error
	InitiateStream(c vo.CircuitID, s vo.StreamID, data []byte) error
	EstablishConnection(c vo.CircuitID, data []byte) error
	TerminateStream(c vo.CircuitID, s vo.StreamID) error
	DestroyCircuit(c vo.CircuitID) error
}

// MessagingServiceFactory produces a CircuitMessagingService bound to a given connection.
type MessagingServiceFactory interface {
	New(conn net.Conn) CircuitMessagingService
}

// TCPMessagingServiceFactory creates TCPCircuitMessagingService instances.
type TCPMessagingServiceFactory struct {
	peSvc PayloadEncodingService
}

// NewTCPMessagingServiceFactory creates a factory with the provided PayloadEncodingService
func NewTCPMessagingServiceFactory(peSvc PayloadEncodingService) MessagingServiceFactory {
	return &TCPMessagingServiceFactory{peSvc: peSvc}
}

// New returns a CircuitMessagingService using the provided connection.
func (f *TCPMessagingServiceFactory) New(conn net.Conn) CircuitMessagingService {
	return NewTCPCircuitMessagingService(conn, f.peSvc)
}

type TCPCircuitMessagingService struct {
	mu    sync.Mutex
	conn  net.Conn
	peSvc PayloadEncodingService
}

// NewTCPCircuitMessagingService wraps an already-connected net.Conn.
// The caller is responsible for establishing the connection.
func NewTCPCircuitMessagingService(conn net.Conn, peSvc PayloadEncodingService) CircuitMessagingService {
	return &TCPCircuitMessagingService{conn: conn, peSvc: peSvc}
}

func (t *TCPCircuitMessagingService) send(cmd vo.CellCommand, cid vo.CircuitID, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cell := entity.Cell{Cmd: cmd, Version: vo.ProtocolV1, Payload: payload}
	buf, err := entity.Encode(cell)
	if err != nil {
		return err
	}
	packet := append(cid.Bytes(), buf...)
	_, err = t.conn.Write(packet)
	return err
}

func (t *TCPCircuitMessagingService) TransmitData(cid vo.CircuitID, s vo.StreamID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	p, err := t.peSvc.EncodeDataPayload(&DataPayloadDTO{StreamID: s.UInt16(), Data: d})
	if err != nil {
		return err
	}
	return t.send(vo.CmdData, cid, p)
}

func (t *TCPCircuitMessagingService) InitiateStream(cid vo.CircuitID, _ vo.StreamID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(vo.CmdBegin, cid, d)
}

func (t *TCPCircuitMessagingService) EstablishConnection(cid vo.CircuitID, d []byte) error {
	if len(d) > entity.MaxPayloadSize {
		return fmt.Errorf("data too big")
	}
	return t.send(vo.CmdConnect, cid, d)
}

func (t *TCPCircuitMessagingService) TerminateStream(cid vo.CircuitID, s vo.StreamID) error {
	p, err := t.peSvc.EncodeDataPayload(&DataPayloadDTO{StreamID: s.UInt16()})
	if err != nil {
		return err
	}
	return t.send(vo.CmdEnd, cid, p)
}

func (t *TCPCircuitMessagingService) DestroyCircuit(cid vo.CircuitID) error {
	return t.send(vo.CmdDestroy, cid, nil)
}
