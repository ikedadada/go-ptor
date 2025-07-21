package value_object

import (
	"crypto/rand"
	"fmt"
)

type AESKey [32]byte

func NewAESKey() (AESKey, error) {
	var k AESKey
	_, err := rand.Read(k[:])
	return k, err
}

func AESKeyFrom(b []byte) (AESKey, error) {
	var k AESKey
	if len(b) != 32 {
		return k, fmt.Errorf("AESKey must be 32B")
	}
	copy(k[:], b)
	return k, nil
}
