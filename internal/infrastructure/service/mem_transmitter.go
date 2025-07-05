package service

import (
	"fmt"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

type MemTransmitter struct {
	Out chan string // デバッグ表示用
}

func NewMemTransmitter(out chan string) service.CircuitTransmitter {
	return &MemTransmitter{
		Out: out,
	}
}

func (tx *MemTransmitter) SendData(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	tx.Out <- fmt.Sprintf("DATA cid=%s sid=%d len=%d", c, s, len(d))
	return nil
}
func (tx *MemTransmitter) SendBegin(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	tx.Out <- fmt.Sprintf("BEGIN cid=%s sid=%d len=%d", c, s, len(d))
	return nil
}
func (tx *MemTransmitter) SendConnect(c value_object.CircuitID, d []byte) error {
	tx.Out <- fmt.Sprintf("CONNECT cid=%s len=%d", c, len(d))
	return nil
}
func (tx *MemTransmitter) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	tx.Out <- fmt.Sprintf("END  cid=%s sid=%d", c, s)
	return nil
}
func (tx *MemTransmitter) SendDestroy(c value_object.CircuitID) error {
	tx.Out <- fmt.Sprintf("DESTROY cid=%s", c)
	return nil
}
