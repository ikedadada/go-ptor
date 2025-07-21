package service_test

import (
	"bytes"
	"testing"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestReadCell(t *testing.T) {
	pcr := service.NewCellReaderService()
	cid := vo.NewCircuitID()
	cellBuf, err := entity.Encode(entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: []byte("hi")})
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
	if gotCell.Cmd != vo.CmdData || string(gotCell.Payload) != "hi" {
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
