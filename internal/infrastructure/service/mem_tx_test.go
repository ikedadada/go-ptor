package service_test

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/service"
	"testing"
)

func TestMemTx_SendData_SendEnd(t *testing.T) {
	ch := make(chan string, 2)

	tx := &service.MemTx{Out: ch}
	cid := value_object.NewCircuitID()
	sid := value_object.NewStreamIDAuto()
	data := []byte("hello")

	err := tx.SendData(cid, sid, data)
	if err != nil {
		t.Fatalf("SendData error: %v", err)
	}
	msg := <-ch
	if msg == "" || msg[:4] != "DATA" {
		t.Errorf("unexpected SendData message: %q", msg)
	}

	err = tx.SendEnd(cid, sid)
	if err != nil {
		t.Fatalf("SendEnd error: %v", err)
	}
	msg = <-ch
	if msg == "" || msg[:3] != "END" {
		t.Errorf("unexpected SendEnd message: %q", msg)
	}
}
