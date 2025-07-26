package value_object

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

var (
	ErrNoPEMData           = errors.New("no PEM data")
	ErrUnsupportedKeyType  = errors.New("unsupported key type")
	ErrUnsupportedPEMBlock = errors.New("unsupported PEM block type")
)

// PrivateKey is an interface for private keys used in the Tor network
type PrivateKey interface {
	// ToPEM encodes the private key to PEM format
	ToPEM() []byte

	// PublicKey returns the corresponding public key
	PublicKey() PublicKey

	// KeyType returns the type of the key (RSA, Ed25519, etc.)
	KeyType() string
}

// ParsePrivateKeyFromPEM parses a PEM-encoded private key and returns either an RSA or Ed25519 private key
func ParsePrivateKeyFromPEM(pemBytes []byte) (PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrNoPEMData
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return NewRSAPrivKey(key), nil

	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		switch k := key.(type) {
		case *rsa.PrivateKey:
			return NewRSAPrivKey(k), nil
		case ed25519.PrivateKey:
			return NewEd25519PrivKey(k), nil
		default:
			return nil, ErrUnsupportedKeyType
		}

	default:
		return nil, ErrUnsupportedPEMBlock
	}
}
