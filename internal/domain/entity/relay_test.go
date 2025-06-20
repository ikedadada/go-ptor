package entity_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"testing"
)

func makeTestRelay() *entity.Relay {
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	pkix, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &pkix.PublicKey
	pkixBytes, _ := x509.MarshalPKIXPublicKey(pub)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes})
	rsaPub, _ := value_object.RSAPubKeyFromPEM(pemBytes)
	end, _ := value_object.NewEndpoint("127.0.0.1", 5000)
	return entity.NewRelay(relayID, end, rsaPub)
}

func TestRelay_Basic(t *testing.T) {
	r := makeTestRelay()
	if r.Status() != entity.Offline {
		t.Errorf("expected Offline")
	}
	r.SetOnline()
	if r.Status() != entity.Online {
		t.Errorf("expected Online")
	}
	r.SetOffline()
	if r.Status() != entity.Offline {
		t.Errorf("expected Offline after SetOffline")
	}
}

func TestRelay_Stats(t *testing.T) {
	r := makeTestRelay()
	r.IncSuccess()
	r.IncSuccess()
	r.IncFailure()
	succ, fail := r.Stats()
	if succ != 2 || fail != 1 {
		t.Errorf("expected 2 success, 1 failure, got %d %d", succ, fail)
	}
}
