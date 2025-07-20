package value_object

import (
	"encoding/base32"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

type HiddenAddr struct{ val string }

func NewHiddenAddr(pub ed25519.PublicKey) HiddenAddr {
	hash := sha3.Sum256(pub)
	addr := base32.StdEncoding.EncodeToString(hash[:])[:52] + ".ptor"
	return HiddenAddr{val: addr}
}

// FromString creates a HiddenAddr from a string address
// This is used when loading addresses from external sources
func HiddenAddrFromString(addr string) HiddenAddr {
	return HiddenAddr{val: addr}
}

func (h HiddenAddr) String() string          { return h.val }
func (h HiddenAddr) Equal(o HiddenAddr) bool { return h.val == o.val }
