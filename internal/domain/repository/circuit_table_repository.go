package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitTableRepository manages per-hop connection states held by a relay.
type CircuitTableRepository interface {
	Add(value_object.CircuitID, *entity.ConnState) error
	Find(value_object.CircuitID) (*entity.ConnState, error)
	Delete(value_object.CircuitID) error
}
