package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

// RSAEncrypt encrypts data using RSA-OAEP with SHA-256.
func RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, in, nil)
}

// RSADecrypt decrypts data using RSA-OAEP with SHA-256.
func RSADecrypt(priv *rsa.PrivateKey, in []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, in, nil)
}

// AESSeal encrypts plaintext with AES-256-GCM.
func AESSeal(key [32]byte, nonce [12]byte, plain []byte) ([]byte, error) {
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

// AESOpen decrypts ciphertext with AES-256-GCM.
func AESOpen(key [32]byte, nonce [12]byte, enc []byte) ([]byte, error) {
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
