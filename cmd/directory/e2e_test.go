package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDirectoryServer(t *testing.T) {
	data := Directory{
		Relays: map[string]RelayInfo{
			"r1": {Endpoint: "127.0.0.1:5000", PubKey: "pk"},
		},
		HiddenServices: map[string]HiddenServiceInfo{
			"h1": {Relay: "r1", PubKey: "hpk"},
		},
	}
	tmp, err := os.CreateTemp(t.TempDir(), "dir.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if err := json.NewEncoder(tmp).Encode(data); err != nil {
		t.Fatalf("encode: %v", err)
	}
	tmp.Close()

	dir, err := loadDirectory(tmp.Name())
	if err != nil {
		t.Fatalf("loadDirectory: %v", err)
	}

	srv := httptest.NewServer(handler(dir))
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer res.Body.Close()

	var got Directory
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Relays["r1"].Endpoint != "127.0.0.1:5000" {
		t.Errorf("relay endpoint mismatch")
	}
	if got.HiddenServices["h1"].Relay != "r1" {
		t.Errorf("hidden relay mismatch")
	}
}
