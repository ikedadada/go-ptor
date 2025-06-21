package value_object

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type RSAPubKey struct{ *rsa.PublicKey }

func RSAPubKeyFromPEM(pemBytes []byte) (RSAPubKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return RSAPubKey{}, errors.New("no PEM data")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return RSAPubKey{}, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return RSAPubKey{}, errors.New("not RSA key")
	}
	return RSAPubKey{PublicKey: rsaPub}, nil
}

func (k RSAPubKey) ToPEM() []byte {
	b := x509.MarshalPKCS1PublicKey(k.PublicKey)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: b})
}
