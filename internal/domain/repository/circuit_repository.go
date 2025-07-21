package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

type CircuitRepository interface {
	Save(*entity.Circuit) error
	Find(vo.CircuitID) (*entity.Circuit, error)
	Delete(vo.CircuitID) error
	ListActive() ([]*entity.Circuit, error)
}
