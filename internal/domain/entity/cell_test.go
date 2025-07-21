package entity_test

import (
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

func TestEncodeDecode(t *testing.T) {
	payload := []byte("hello")
	c := entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}
	buf, err := entity.Encode(c)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(buf) != entity.MaxCellSize {
		t.Fatalf("size: %d", len(buf))
	}
	d, err := entity.Decode(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Cmd != c.Cmd || d.Version != c.Version || string(d.Payload) != string(payload) {
		t.Fatalf("mismatch")
	}
}
