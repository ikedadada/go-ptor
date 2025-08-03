package usecase_test

import (
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

func TestHandleEndUseCase(t *testing.T) {
	cir, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	ss, err := cir.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	sid := ss.ID
	cid := cir.ID()

	tests := []struct {
		name         string
		circuitID    vo.CircuitID
		streamID     vo.StreamID
		setupMock    func(repo repository.CircuitRepository)
		expectsError bool
	}{
		{
			name:      "stream",
			circuitID: cid,
			streamID:  sid,
			setupMock: func(repo repository.CircuitRepository) {
				WhenDouble(repo.Find(cir.ID())).ThenReturn(cir, nil)
			},
			expectsError: false,
		},
		{
			name:      "circuit",
			circuitID: cid,
			streamID:  0,
			setupMock: func(repo repository.CircuitRepository) {
				WhenDouble(repo.Find(cir.ID())).ThenReturn(cir, nil)
				WhenSingle(repo.Delete(cir.ID())).ThenReturn(nil)
			},
			expectsError: false,
		},
		{
			name:      "not found",
			circuitID: cid,
			streamID:  sid,
			setupMock: func(repo repository.CircuitRepository) {
				WhenDouble(repo.Find(cir.ID())).ThenReturn(nil, repository.ErrNotFound)
			},
			expectsError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)
			repo := Mock[repository.CircuitRepository](ctrl)
			tt.setupMock(repo)

			uc := usecase.NewHandleEndUseCase(repo)
			err := uc.Handle(usecase.HandleEndInput{CircuitID: tt.circuitID, StreamID: tt.streamID})

			if tt.expectsError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectsError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectsError {
				// For circuit deletion test, verify Delete was called
				if tt.streamID == 0 {
					Verify(repo, Times(1)).Delete(cir.ID())
				}
			}
		})
	}
}
