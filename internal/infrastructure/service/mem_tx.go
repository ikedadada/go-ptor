package service

import (
	"fmt"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type MemTx struct {
	Out chan string // デバッグ表示用
}

func (tx *MemTx) SendData(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	tx.Out <- fmt.Sprintf("DATA cid=%s sid=%d len=%d", c, s, len(d))
	return nil
}
func (tx *MemTx) SendEnd(c value_object.CircuitID, s value_object.StreamID) error {
	tx.Out <- fmt.Sprintf("END  cid=%s sid=%d", c, s)
	return nil
}
