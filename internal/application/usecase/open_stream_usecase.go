package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type OpenStreamInput struct {
	CircuitID string
}

type OpenStreamOutput struct {
	CircuitID string `json:"circuit_id"`
	StreamID  uint16 `json:"stream_id"`
}

type OpenStreamUseCase interface {
	Handle(in OpenStreamInput) (OpenStreamOutput, error)
}

// OpenStreamInteractor
type openStreamUseCaseImpl struct {
	cr repository.CircuitRepository
}

func NewOpenStreamInteractor(cr repository.CircuitRepository) *openStreamUseCaseImpl {
	return &openStreamUseCaseImpl{cr: cr}
}

// Handle: 既存 Circuit を取得 → StreamState 生成 → DTO で返却
func (uc *openStreamUseCaseImpl) Handle(in OpenStreamInput) (OpenStreamOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
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
