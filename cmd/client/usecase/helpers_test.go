package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func makeTestCircuit() (*entity.Circuit, error) {
	id, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	key, err := vo.NewAESKey()
	if err != nil {
		return nil, err
	}
	nonce, err := vo.NewNonce()
	if err != nil {
		return nil, err
	}
	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	priv := vo.NewRSAPrivKey(rawKey)
	c, err := entity.NewCircuit(id, []vo.RelayID{relayID}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	return c, nil
}
