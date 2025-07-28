package usecase

import (
	"testing"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestDecryptCellDataUseCase_Handle_DestroyCell(t *testing.T) {
	// Create mock cell
	cell, err := entity.NewCell(vo.CmdDestroy, []byte("test payload"))
	if err != nil {
		t.Fatalf("NewCell: %v", err)
	}

	// Create mock circuit
	circuit := &entity.Circuit{}

	// Create use case with nil services (won't be called for destroy cell)
	uc := NewDecryptCellDataUseCase(nil, nil)

	// Test
	result, err := uc.Handle(DecryptCellDataInput{
		Cell:    cell,
		Circuit: circuit,
	})

	// Assertions
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !result.ShouldClose {
		t.Error("Expected ShouldClose to be true for destroy cell")
	}
	if result.CellData != nil {
		t.Error("Expected CellData to be nil for destroy cell")
	}
}

func TestDecryptCellDataUseCase_Handle_UnhandledCommand(t *testing.T) {
	// Create mock cell with unhandled command
	cell, err := entity.NewCell(vo.CmdBegin, []byte("test payload"))
	if err != nil {
		t.Fatalf("NewCell: %v", err)
	}

	// Create mock circuit
	circuit := &entity.Circuit{}

	// Create use case with nil services (won't be called for unhandled command)
	uc := NewDecryptCellDataUseCase(nil, nil)

	// Test
	result, err := uc.Handle(DecryptCellDataInput{
		Cell:    cell,
		Circuit: circuit,
	})

	// Assertions
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result.ShouldClose {
		t.Error("Expected ShouldClose to be false for unhandled command")
	}
	if result.CellData != nil {
		t.Error("Expected CellData to be nil for unhandled command")
	}
}
