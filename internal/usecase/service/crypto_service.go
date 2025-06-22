package service

import "crypto/rsa"

// CryptoService provides common cryptographic operations used by the application.
type CryptoService interface {
	RSAEncrypt(pub *rsa.PublicKey, in []byte) ([]byte, error)
	RSADecrypt(priv *rsa.PrivateKey, in []byte) ([]byte, error)
	AESSeal(key [32]byte, nonce [12]byte, plain []byte) ([]byte, error)
	AESOpen(key [32]byte, nonce [12]byte, enc []byte) ([]byte, error)
	// AESMultiSeal applies AESSeal repeatedly with the given keys and nonces
	// from last to first, producing an onion-encrypted payload.
	AESMultiSeal(keys [][32]byte, nonces [][12]byte, plain []byte) ([]byte, error)
	// AESMultiOpen decrypts an onion-encrypted payload by sequentially
	// applying AESOpen with each key and nonce in order.
	AESMultiOpen(keys [][32]byte, nonces [][12]byte, enc []byte) ([]byte, error)
}
