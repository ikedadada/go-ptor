package value_object_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func TestRSAPubKey_Table(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &key.PublicKey
	pkixBytes, _ := x509.MarshalPKIXPublicKey(pub)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes})
	tests := []struct {
		name       string
		input      []byte
		expectsErr bool
	}{
		{"valid pem", pemBytes, false},
		{"invalid pem", []byte("notpem"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value_object.RSAPubKeyFromPEM(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}
