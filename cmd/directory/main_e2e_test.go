package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestDirectoryServer(t *testing.T) {
	storage := directoryStorage{
		relays: []relayDTO{
			{ID: "r1", Endpoint: "127.0.0.1:5000", PubKey: "pk"},
		},
		hiddenServices: []hiddenServiceDTO{
			{Address: "h1", Relay: "r1", PubKey: "hpk"},
		},
	}

	srv := httptest.NewServer(newMux(storage))
	defer srv.Close()

	// Test /relays endpoint
	res, err := srv.Client().Get(srv.URL + "/relays")
	if err != nil {
		t.Fatalf("http get relays: %v", err)
	}
	defer res.Body.Close()

	var gotRelays []relayDTO
	if err := json.NewDecoder(res.Body).Decode(&gotRelays); err != nil {
		t.Fatalf("decode relays: %v", err)
	}
	if len(gotRelays) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(gotRelays))
	}
	if gotRelays[0].Endpoint != "127.0.0.1:5000" {
		t.Errorf("relay endpoint mismatch, got %s, want 127.0.0.1:5000", gotRelays[0].Endpoint)
	}
	if gotRelays[0].PubKey != "pk" {
		t.Errorf("relay pubkey mismatch, got %s, want pk", gotRelays[0].PubKey)
	}

	// Test /hidden_services endpoint
	res2, err := srv.Client().Get(srv.URL + "/hidden_services")
	if err != nil {
		t.Fatalf("http get hidden services: %v", err)
	}
	defer res2.Body.Close()

	var gotHidden []hiddenServiceDTO
	if err := json.NewDecoder(res2.Body).Decode(&gotHidden); err != nil {
		t.Fatalf("decode hidden services: %v", err)
	}
	if len(gotHidden) != 1 {
		t.Fatalf("expected 1 hidden service, got %d", len(gotHidden))
	}
	if gotHidden[0].Relay != "r1" {
		t.Errorf("hidden service relay mismatch, got %s, want r1", gotHidden[0].Relay)
	}
	if gotHidden[0].PubKey != "hpk" {
		t.Errorf("hidden service pubkey mismatch, got %s, want hpk", gotHidden[0].PubKey)
	}
}
