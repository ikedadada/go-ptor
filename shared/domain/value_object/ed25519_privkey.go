package value_object

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

type Ed25519PrivKey struct {
	key ed25519.PrivateKey
}

// Ensure Ed25519PrivKey implements PrivateKey interface
var _ PrivateKey = (*Ed25519PrivKey)(nil)

// NewEd25519PrivKey creates a new Ed25519PrivKey value object
func NewEd25519PrivKey(key ed25519.PrivateKey) *Ed25519PrivKey {
	if key == nil {
		return nil
	}
	return &Ed25519PrivKey{key: key}
}

// ToPEM encodes the Ed25519 private key to PEM format
func (k *Ed25519PrivKey) ToPEM() []byte {
	if k == nil || k.key == nil {
		return nil
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(k.key)
	if err != nil {
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})
}

// PublicKey returns the corresponding Ed25519 public key
func (k *Ed25519PrivKey) PublicKey() PublicKey {
	if k == nil || k.key == nil {
		return nil
	}
	return Ed25519PubKey{PublicKey: k.key.Public().(ed25519.PublicKey)}
}

// KeyType returns the key type
func (k *Ed25519PrivKey) KeyType() string {
	if k == nil {
		return ""
	}
	return "Ed25519"
}

// Ed25519Key returns the underlying ed25519.PrivateKey for interoperability
// This method should be used sparingly and only when necessary for crypto operations
func (k *Ed25519PrivKey) Ed25519Key() ed25519.PrivateKey {
	if k == nil {
		return nil
	}
	return k.key
}

// Ed25519PrivKeyFromPEM creates an Ed25519PrivKey from PEM-encoded bytes
func Ed25519PrivKeyFromPEM(pemBytes []byte) (*Ed25519PrivKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrNoPEMData
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ed25519Key, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: expected ed25519.PrivateKey, but got %T", ErrUnsupportedKeyType, key)
	}

	return NewEd25519PrivKey(ed25519Key), nil
}
