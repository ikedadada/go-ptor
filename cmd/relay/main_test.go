package main

import (
	"bytes"
	"encoding/binary"
	"github.com/google/uuid"
	"testing"
)

func TestReadCell_Data(t *testing.T) {
	cid := uuid.New()
	sid := uint16(10)
	data := []byte("abc")
	buf := new(bytes.Buffer)
	buf.Write(cid[:])
	binary.Write(buf, binary.BigEndian, sid)
	binary.Write(buf, binary.BigEndian, uint16(len(data)))
	buf.Write(data)

	c, err := readCell(buf)
	if err != nil {
		t.Fatalf("readCell error: %v", err)
	}
	if c.circID.String() != cid.String() {
		t.Errorf("cid mismatch")
	}
	if c.streamID.UInt16() != sid {
		t.Errorf("sid mismatch")
	}
	if !bytes.Equal(c.data, data) {
		t.Errorf("data mismatch")
	}
	if c.end {
		t.Errorf("unexpected end flag")
	}
}

func TestReadCell_End(t *testing.T) {
	cid := uuid.New()
	sid := uint16(5)
	buf := new(bytes.Buffer)
	buf.Write(cid[:])
	binary.Write(buf, binary.BigEndian, sid)
	binary.Write(buf, binary.BigEndian, uint16(0xFFFF))

	c, err := readCell(buf)
	if err != nil {
		t.Fatalf("readCell error: %v", err)
	}
	if !c.end {
		t.Errorf("expected end flag")
	}
	if len(c.data) != 0 {
		t.Errorf("expected no data")
	}
}
