package value_object

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

type RSAPrivKey struct {
	key *rsa.PrivateKey
}

// Ensure RSAPrivKey implements PrivateKey interface
var _ PrivateKey = (*RSAPrivKey)(nil)

// NewRSAPrivKey creates a new RSAPrivKey value object
func NewRSAPrivKey(key *rsa.PrivateKey) *RSAPrivKey {
	if key == nil {
		return nil
	}
	return &RSAPrivKey{key: key}
}

// ToPEM encodes the RSA private key to PEM format
func (k *RSAPrivKey) ToPEM() []byte {
	if k == nil || k.key == nil {
		return nil
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(k.key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})
}

// PublicKey returns the corresponding RSA public key
func (k *RSAPrivKey) PublicKey() PublicKey {
	if k == nil || k.key == nil {
		return nil
	}
	return RSAPubKey{PublicKey: &k.key.PublicKey}
}

// KeyType returns the key type
func (k *RSAPrivKey) KeyType() string {
	if k == nil {
		return ""
	}
	return "RSA"
}

// RSAKey returns the underlying *rsa.PrivateKey for interoperability
// This method should be used sparingly and only when necessary for crypto operations
func (k *RSAPrivKey) RSAKey() *rsa.PrivateKey {
	if k == nil {
		return nil
	}
	return k.key
}

// RSAPrivKeyFromPEM creates an RSAPrivKey from PEM-encoded bytes
func RSAPrivKeyFromPEM(pemBytes []byte) (*RSAPrivKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrNoPEMData
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return NewRSAPrivKey(key), nil
}
