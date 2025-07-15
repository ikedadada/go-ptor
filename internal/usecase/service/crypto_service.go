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

	// X25519Generate returns a new private/public key pair for X25519.
	X25519Generate() (priv, pub []byte, err error)
	// X25519Shared derives a shared secret between priv and pub.
	X25519Shared(priv, pub []byte) ([]byte, error)
	// DeriveKeyNonce expands the shared secret into an AES key and nonce.
	DeriveKeyNonce(secret []byte) ([32]byte, [12]byte, error)

	// ModifyNonceWithSequence creates a unique nonce by XORing sequence number into base nonce
	ModifyNonceWithSequence(baseNonce [12]byte, sequence uint64) [12]byte
}
