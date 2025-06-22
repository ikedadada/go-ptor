package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
)

func TestDirectoryServer(t *testing.T) {
	data := entity.Directory{
		Relays: map[string]entity.RelayInfo{
			"r1": {Endpoint: "127.0.0.1:5000", PubKey: "pk"},
		},
		HiddenServices: map[string]entity.HiddenServiceInfo{
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

	srv := httptest.NewServer(newMux(dir))
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL + "/relays.json")
	if err != nil {
		t.Fatalf("http get relays: %v", err)
	}
	defer res.Body.Close()

	var got entity.Directory
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Relays["r1"].Endpoint != "127.0.0.1:5000" {
		t.Errorf("relay endpoint mismatch")
	}

	res2, err := srv.Client().Get(srv.URL + "/hidden.json")
	if err != nil {
		t.Fatalf("http get hidden: %v", err)
	}
	defer res2.Body.Close()

	var got2 entity.Directory
	if err := json.NewDecoder(res2.Body).Decode(&got2); err != nil {
		t.Fatalf("decode hidden: %v", err)
	}
	if got2.HiddenServices["h1"].Relay != "r1" {
		t.Errorf("hidden relay mismatch")
	}
}
