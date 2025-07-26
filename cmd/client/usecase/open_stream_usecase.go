package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// OpenStreamInput specifies which circuit to open a new stream for.
type OpenStreamInput struct {
	CircuitID string
}

// OpenStreamOutput holds the created stream identifier.
type OpenStreamOutput struct {
	CircuitID string `json:"circuit_id"`
	StreamID  uint16 `json:"stream_id"`
}

// OpenStreamUseCase opens a new stream on an existing circuit.
type OpenStreamUseCase interface {
	Handle(in OpenStreamInput) (OpenStreamOutput, error)
}

// OpenStreamInteractor
type openStreamUseCaseImpl struct {
	cr repository.CircuitRepository
}

// NewOpenStreamUseCase returns a use case to open streams on circuits.
func NewOpenStreamUseCase(cr repository.CircuitRepository) OpenStreamUseCase {
	return &openStreamUseCaseImpl{cr: cr}
}

// Handle: 既存 Circuit を取得 → StreamState 生成 → DTO で返却
func (uc *openStreamUseCaseImpl) Handle(in OpenStreamInput) (OpenStreamOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return OpenStreamOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}

	cir, err := uc.cr.Find(cid)
	if err != nil {
		return OpenStreamOutput{}, fmt.Errorf("circuit not found: %w", err)
	}

	st, err := cir.OpenStream()
	if err != nil {
		return OpenStreamOutput{}, fmt.Errorf("open stream: %w", err)
	}

	return OpenStreamOutput{
		CircuitID: cir.ID().String(),
		StreamID:  st.ID.UInt16(),
	}, nil
}
