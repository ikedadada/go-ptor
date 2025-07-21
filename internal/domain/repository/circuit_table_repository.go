package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

// CircuitTableRepository manages per-hop connection states held by a relay.
type CircuitTableRepository interface {
	Add(vo.CircuitID, *entity.ConnState) error
	Find(vo.CircuitID) (*entity.ConnState, error)
	Delete(vo.CircuitID) error
}
