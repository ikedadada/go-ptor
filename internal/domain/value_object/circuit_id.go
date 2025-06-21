package value_object

import "github.com/google/uuid"

type CircuitID struct{ val uuid.UUID }

func NewCircuitID() CircuitID { return CircuitID{uuid.New()} }
func CircuitIDFrom(s string) (CircuitID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return CircuitID{}, err
	}
	return CircuitID{val: id}, nil
}
func (c CircuitID) String() string         { return c.val.String() }
func (c CircuitID) Equal(o CircuitID) bool { return c.val == o.val }
func (c CircuitID) Bytes() []byte {
	return c.val[:]
}
