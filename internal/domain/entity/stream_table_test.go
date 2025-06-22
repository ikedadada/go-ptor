package entity_test

import (
	"net"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

func TestStreamTable_AddGetRemove(t *testing.T) {
	tbl := entity.NewStreamTable()
	id := value_object.NewStreamIDAuto()
	c1, c2 := net.Pipe()
	defer c2.Close()

	if err := tbl.Add(id, c1); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := tbl.Add(id, c2); err != entity.ErrDuplicate {
		t.Fatalf("expect ErrDuplicate")
	}
	got, err := tbl.Get(id)
	if err != nil || got != c1 {
		t.Fatalf("get: %v", err)
	}
	if err := tbl.Remove(id); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := tbl.Get(id); err != entity.ErrNotFound {
		t.Fatalf("expected ErrNotFound")
	}
}

func TestStreamTable_DestroyAll(t *testing.T) {
	tbl := entity.NewStreamTable()
	id1 := value_object.NewStreamIDAuto()
	id2 := value_object.NewStreamIDAuto()
	c1, _ := net.Pipe()
	c2, _ := net.Pipe()

	_ = tbl.Add(id1, c1)
	_ = tbl.Add(id2, c2)
	tbl.DestroyAll()

	if _, err := tbl.Get(id1); err != entity.ErrNotFound {
		t.Fatalf("table not cleared")
	}
	if _, err := tbl.Get(id2); err != entity.ErrNotFound {
		t.Fatalf("table not cleared")
	}
}
