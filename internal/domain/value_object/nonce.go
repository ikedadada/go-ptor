package value_object

import (
	"crypto/rand"
	"fmt"
)

type Nonce [12]byte

func NewNonce() (Nonce, error) {
	var n Nonce
	_, err := rand.Read(n[:])
	return n, err
}

func NonceFrom(b []byte) (Nonce, error) {
	var n Nonce
	if len(b) != 12 {
		return n, fmt.Errorf("nonce must be 12B")
	}
	copy(n[:], b)
	return n, nil
}
