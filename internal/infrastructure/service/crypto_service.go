package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"

	"ikedadada/go-ptor/internal/usecase/service"
)

// CryptoServiceImpl implements service.CryptoService using the standard library.
type cryptoServiceImpl struct{}

// NewCryptoService returns a CryptoService backed by Go's crypto packages.
func NewCryptoService() service.CryptoService { return &cryptoServiceImpl{} }

func (*cryptoServiceImpl) RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, in, nil)
}

func (*cryptoServiceImpl) RSADecrypt(priv *rsa.PrivateKey, in []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, in, nil)
}

func (*cryptoServiceImpl) AESSeal(key [32]byte, nonce [12]byte, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce[:], plain, nil), nil
}

func (*cryptoServiceImpl) AESOpen(key [32]byte, nonce [12]byte, enc []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce[:], enc, nil)
}
