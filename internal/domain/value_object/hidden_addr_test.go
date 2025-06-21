package value_object_test

import (
	"crypto/rand"
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"

	"golang.org/x/crypto/ed25519"
)

func TestHiddenAddr_Table(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tests := []struct {
		name string
		pub  ed25519.PublicKey
	}{
		{"valid pubkey", pub},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := value_object.NewHiddenAddr(tt.pub)
			if len(h.String()) != 57 || h.String()[52:] != ".ptor" {
				t.Errorf("unexpected hidden addr: %s", h.String())
			}
		})
	}
}
