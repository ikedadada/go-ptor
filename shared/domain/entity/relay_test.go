package entity_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"testing"
)

func makeTestRelay() (*entity.Relay, error) {
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	pkix, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pub := &pkix.PublicKey
	pkixBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes})
	rsaPub, err := vo.RSAPubKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	end, err := vo.NewEndpoint("127.0.0.1", 5000)
	if err != nil {
		return nil, err
	}
	return entity.NewRelay(relayID, end, rsaPub), nil
}

func TestRelay_Basic(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}

	tests := []struct {
		name   string
		action func()
		want   entity.RelayStatus
	}{
		{"initial", func() {}, entity.Offline},
		{"online", relay.SetOnline, entity.Online},
		{"offline", relay.SetOffline, entity.Offline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.action()
			if relay.Status() != tt.want {
				t.Errorf("expected %v, got %v", tt.want, relay.Status())
			}
		})
	}
}

func TestRelay_Stats(t *testing.T) {
	relay, err := makeTestRelay()
	if err != nil {
		t.Fatalf("setup relay: %v", err)
	}

	relay.IncSuccess()
	relay.IncSuccess()
	relay.IncFailure()

	tests := []struct {
		name     string
		wantSucc uint64
		wantFail uint64
	}{
		{"stats", 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			succ, fail := relay.Stats()
			if succ != tt.wantSucc || fail != tt.wantFail {
				t.Errorf("expected %d success, %d failure, got %d %d", tt.wantSucc, tt.wantFail, succ, fail)
			}
		})
	}
}
