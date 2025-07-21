package value_object

import (
	"fmt"

	"github.com/google/uuid"
)

// RelayID は UUIDv4 に限定
type RelayID struct {
	val uuid.UUID
}

func NewRelayID(s string) (RelayID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return RelayID{}, err
	}
	if id.Version() != 4 {
		return RelayID{}, fmt.Errorf("relay id must be uuid v4, got v%d", id.Version())
	}
	return RelayID{val: id}, nil
}

func (r RelayID) String() string       { return r.val.String() }
func (r RelayID) Equal(o RelayID) bool { return r.val == o.val }
