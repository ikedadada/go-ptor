package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

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

func (c *cryptoServiceImpl) AESMultiSeal(keys [][32]byte, nonces [][12]byte, plain []byte) ([]byte, error) {
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys/nonces length mismatch")
	}
	out := make([]byte, len(plain))
	copy(out, plain)
	var err error
	for i := len(keys) - 1; i >= 0; i-- {
		out, err = c.AESSeal(keys[i], nonces[i], out)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *cryptoServiceImpl) AESMultiOpen(keys [][32]byte, nonces [][12]byte, enc []byte) ([]byte, error) {
	if len(keys) != len(nonces) {
		return nil, fmt.Errorf("keys/nonces length mismatch")
	}
	out := enc
	var err error
	for i := 0; i < len(keys); i++ {
		out, err = c.AESOpen(keys[i], nonces[i], out)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (*cryptoServiceImpl) X25519Generate() (priv, pub []byte, err error) {
	curve := ecdh.X25519()
	kp, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return kp.Bytes(), kp.PublicKey().Bytes(), nil
}

func (*cryptoServiceImpl) X25519Shared(privBytes, pubBytes []byte) ([]byte, error) {
	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}
	pub, err := curve.NewPublicKey(pubBytes)
	if err != nil {
		return nil, err
	}
	return priv.ECDH(pub)
}

func (*cryptoServiceImpl) DeriveKeyNonce(secret []byte) ([32]byte, [12]byte, error) {
	var key [32]byte
	var nonce [12]byte
	hk := hkdf.New(sha256.New, secret, nil, []byte("go-ptor"))
	if _, err := io.ReadFull(hk, key[:]); err != nil {
		return key, nonce, err
	}
	// Generate random nonce instead of deriving from secret to avoid reuse
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return key, nonce, err
	}
	return key, nonce, nil
}
