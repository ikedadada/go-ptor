package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type RelayRepository interface {
	Save(*entity.Relay) error
	FindByID(value_object.RelayID) (*entity.Relay, error)
	AllOnline() ([]*entity.Relay, error)
}
