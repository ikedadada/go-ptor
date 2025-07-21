package value_object_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"testing"
)

func TestRSAPubKey_Table(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pub := &key.PublicKey
	pkixBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
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
			_, err := vo.RSAPubKeyFromPEM(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}
