package value_object

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type Ed25519PubKey struct{ ed25519.PublicKey }

// Ensure Ed25519PubKey implements PublicKey interface
var _ PublicKey = Ed25519PubKey{}

func Ed25519PubKeyFromPEM(pemBytes []byte) (Ed25519PubKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return Ed25519PubKey{}, errors.New("no PEM data")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return Ed25519PubKey{}, err
	}
	ed25519Pub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return Ed25519PubKey{}, errors.New("not Ed25519 key")
	}
	return Ed25519PubKey{PublicKey: ed25519Pub}, nil
}

func (k Ed25519PubKey) ToPEM() []byte {
	b, _ := x509.MarshalPKIXPublicKey(k.PublicKey)
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})
}