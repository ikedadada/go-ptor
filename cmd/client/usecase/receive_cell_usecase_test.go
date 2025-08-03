package usecase_test

import (
	"errors"
	"io"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

func TestReceiveCellUseCase_Handle(t *testing.T) {
	// Test data setup
	cir, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	circuitID := cir.ID()
	testCell := &entity.Cell{
		Cmd:     vo.CmdData,
		Payload: []byte("test data"),
	}

	tests := []struct {
		name               string
		input              usecase.ReceiveCellInput
		setupMock          func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit)
		expectedIsEOF      bool
		expectedHasCell    bool
		expectedHasCircuit bool
		expectsError       bool
	}{
		{
			name: "successful cell receive",
			input: usecase.ReceiveCellInput{
				CircuitID: circuitID,
			},
			setupMock: func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit) {
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(cir, nil)
				When(crSvc.ReadCell(Any[io.Reader]())).ThenReturn(vo.CircuitID{}, testCell, nil)
			},
			expectedIsEOF:      false,
			expectedHasCell:    true,
			expectedHasCircuit: true,
			expectsError:       false,
		},
		{
			name: "EOF error",
			input: usecase.ReceiveCellInput{
				CircuitID: circuitID,
			},
			setupMock: func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit) {
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(cir, nil)
				When(crSvc.ReadCell(Any[io.Reader]())).ThenReturn(vo.CircuitID{}, nil, io.EOF)
			},
			expectedIsEOF:      true,
			expectedHasCell:    false,
			expectedHasCircuit: false,
			expectsError:       false,
		},
		{
			name: "circuit not found",
			input: usecase.ReceiveCellInput{
				CircuitID: circuitID,
			},
			setupMock: func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit) {
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(nil, repository.ErrNotFound)
			},
			expectedIsEOF:      false,
			expectedHasCell:    false,
			expectedHasCircuit: false,
			expectsError:       true,
		},
		{
			name: "no connection for circuit",
			input: usecase.ReceiveCellInput{
				CircuitID: circuitID,
			},
			setupMock: func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit) {
				// Create circuit without connection
				emptyCircuit, _ := makeTestCircuit()
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(emptyCircuit, nil)
			},
			expectedIsEOF:      false,
			expectedHasCell:    false,
			expectedHasCircuit: false,
			expectsError:       true,
		},
		{
			name: "read cell error",
			input: usecase.ReceiveCellInput{
				CircuitID: circuitID,
			},
			setupMock: func(cRepo repository.CircuitRepository, crSvc service.CellReaderService, cir *entity.Circuit) {
				WhenDouble(cRepo.Find(circuitID)).ThenReturn(cir, nil)
				When(crSvc.ReadCell(Any[io.Reader]())).ThenReturn(vo.CircuitID{}, nil, errors.New("read error"))
			},
			expectedIsEOF:      false,
			expectedHasCell:    false,
			expectedHasCircuit: false,
			expectsError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			mockConn := Mock[net.Conn](ctrl)
			cir.SetConn(0, mockConn)
			cRepo := Mock[repository.CircuitRepository](ctrl)
			crSvc := Mock[service.CellReaderService](ctrl)

			tt.setupMock(cRepo, crSvc, cir)

			uc := usecase.NewReceiveCellUseCase(cRepo, crSvc)
			result, err := uc.Handle(tt.input)

			if tt.expectsError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if result.IsEOF != tt.expectedIsEOF {
					t.Errorf("expected IsEOF %v, got %v", tt.expectedIsEOF, result.IsEOF)
				}

				if tt.expectedHasCell && result.Cell == nil {
					t.Errorf("expected cell but got nil")
				}
				if !tt.expectedHasCell && result.Cell != nil {
					t.Errorf("expected no cell but got one")
				}

				if tt.expectedHasCircuit && result.Circuit == nil {
					t.Errorf("expected circuit but got nil")
				}
				if !tt.expectedHasCircuit && result.Circuit != nil {
					t.Errorf("expected no circuit but got one")
				}
			}
		})
	}
}
