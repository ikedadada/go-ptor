package value_object

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// PublicKey is an interface for public keys used in hidden services
type PublicKey interface {
	ToPEM() []byte
}

// ParsePublicKeyFromPEM parses a PEM-encoded public key and returns either an RSA or Ed25519 public key
func ParsePublicKeyFromPEM(pemBytes []byte) (PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM data")
	}
	
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	switch key := pub.(type) {
	case *rsa.PublicKey:
		return RSAPubKey{PublicKey: key}, nil
	case ed25519.PublicKey:
		return Ed25519PubKey{PublicKey: key}, nil
	default:
		return nil, errors.New("unsupported public key type")
	}
}