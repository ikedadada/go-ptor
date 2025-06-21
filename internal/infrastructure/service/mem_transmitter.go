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
func (tx *MemTransmitter) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	tx.Out <- fmt.Sprintf("END  cid=%s sid=%d", c, s)
	return nil
}
