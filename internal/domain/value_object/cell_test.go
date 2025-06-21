package value_object_test

import (
	"testing"

	valueobject "ikedadada/go-ptor/internal/domain/value_object"
)

func TestEncodeDecode(t *testing.T) {
	payload := []byte("hello")
	c := valueobject.Cell{Cmd: valueobject.CmdData, Version: valueobject.Version, Payload: payload}
	buf, err := valueobject.Encode(c)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(buf) != valueobject.MaxCellSize {
		t.Fatalf("size: %d", len(buf))
	}
	d, err := valueobject.Decode(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Cmd != c.Cmd || d.Version != c.Version || string(d.Payload) != string(payload) {
		t.Fatalf("mismatch")
	}
}
