package service

import "crypto/rsa"

// CryptoService provides common cryptographic operations used by the application.
type CryptoService interface {
	RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error)
	RSADecrypt(priv *rsa.PrivateKey, in []byte) ([]byte, error)
	AESSeal(key [32]byte, nonce [12]byte, plain []byte) ([]byte, error)
	AESOpen(key [32]byte, nonce [12]byte, enc []byte) ([]byte, error)
}
