package service

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service" // 依存関係のために必要
)

// 簡易セル: | CID(16B) | SID(2B) | LEN(2B) | DATA |
const hdr = 20 // 16+2+2

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

func (t *TCPTransmitter) send(cid value_object.CircuitID, sid value_object.StreamID, data []byte, flag byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	buf := make([]byte, hdr+len(data))
	copy(buf[:16], cid.Bytes())
	binary.BigEndian.PutUint16(buf[16:18], sid.UInt16())
	binary.BigEndian.PutUint16(buf[18:20], uint16(len(data)))
	copy(buf[20:], data)
	if flag != 0 {
		buf[18] = flag // 特殊フラグ
	}
	_, err := t.conn.Write(buf)
	return err
}

func (t *TCPTransmitter) SendData(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	if len(d) > 65535 {
		return fmt.Errorf("data too big")
	}
	return t.send(c, s, d, 0)
}

func (t *TCPTransmitter) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	return t.send(c, s, nil, 0xFF)
}

func (t *TCPTransmitter) SendDestroy(c value_object.CircuitID) error {
	return t.send(c, 0, nil, 0xFE)
}
