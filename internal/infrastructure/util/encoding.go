package util

import (
	"bytes"
	"encoding/gob"
)

// EncodePayload serializes any type using gob encoding
func EncodePayload(payload any) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(payload)
	return buf.Bytes(), err
}

// DecodePayload deserializes bytes into the specified type using generics
func DecodePayload[T any](data []byte) (*T, error) {
	var result T
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&result)
	return &result, err
}