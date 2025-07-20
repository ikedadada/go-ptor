package service_test

import (
	"bytes"
	"testing"

	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

func TestReadCell(t *testing.T) {
	pcr := service.NewCellReaderService()
	cid := value_object.NewCircuitID()
	cellBuf, err := value_object.Encode(value_object.Cell{Cmd: value_object.CmdData, Version: value_object.ProtocolV1, Payload: []byte("hi")})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	buf := append(cid.Bytes(), cellBuf...)
	gotCID, gotCell, err := pcr.ReadCell(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !cid.Equal(gotCID) {
		t.Fatalf("cid mismatch")
	}
	if gotCell.Cmd != value_object.CmdData || string(gotCell.Payload) != "hi" {
		t.Fatalf("cell mismatch")
	}
}

func TestReadCell_Err(t *testing.T) {
	pcr := service.NewCellReaderService()
	_, _, err := pcr.ReadCell(bytes.NewReader([]byte("short")))
	if err == nil {
		t.Fatalf("expected error")
	}
}
