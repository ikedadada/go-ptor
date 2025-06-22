package service_test

import (
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/infrastructure/service"
	"testing"
)

func TestMemTx_SendData_SendEnd_Destroy(t *testing.T) {
	ch := make(chan string, 4)

	tx := &service.MemTransmitter{Out: ch}
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

	err = tx.SendBegin(cid, sid, data)
	if err != nil {
		t.Fatalf("SendBegin error: %v", err)
	}
	msg = <-ch
	if msg == "" || msg[:5] != "BEGIN" {
		t.Errorf("unexpected SendBegin message: %q", msg)
	}

	err = tx.SendEnd(cid, sid)
	if err != nil {
		t.Fatalf("SendEnd error: %v", err)
	}
	msg = <-ch
	if msg == "" || msg[:3] != "END" {
		t.Errorf("unexpected SendEnd message: %q", msg)
	}

	err = tx.SendDestroy(cid)
	if err != nil {
		t.Fatalf("SendDestroy error: %v", err)
	}
	msg = <-ch
	if msg == "" || msg[:7] != "DESTROY" {
		t.Errorf("unexpected SendDestroy message: %q", msg)
	}
}
