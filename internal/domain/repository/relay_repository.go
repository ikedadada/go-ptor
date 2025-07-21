package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

type RelayRepository interface {
	Save(*entity.Relay) error
	FindByID(vo.RelayID) (*entity.Relay, error)
	AllOnline() ([]*entity.Relay, error)
}
