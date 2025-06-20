package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type CircuitRepository interface {
	Save(*entity.Circuit) error
	Find(value_object.CircuitID) (*entity.Circuit, error)
	Delete(value_object.CircuitID) error
	ListActive() ([]*entity.Circuit, error)
}
