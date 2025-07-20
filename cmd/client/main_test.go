package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/usecase"
)

func TestDirectoryServiceUseCase_NormalizesKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/hidden.json" {
			json.NewEncoder(w).Encode(entity.Directory{HiddenServices: map[string]entity.HiddenServiceInfo{
				"UPPER.PTOR": {Relay: "r1", PubKey: "pk"},
			}})
		} else if r.URL.Path == "/relays.json" {
			json.NewEncoder(w).Encode(entity.Directory{Relays: map[string]entity.RelayInfo{}})
		}
	}))
	defer srv.Close()

	directoryUC := usecase.NewDirectoryServiceUseCase()
	out, err := directoryUC.FetchHiddenServices(usecase.DirectoryServiceInput{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("FetchHiddenServices: %v", err)
	}
	got := out.HiddenServices
	if _, ok := got["upper.ptor"]; !ok {
		t.Fatalf("normalized key not found")
	}
	if _, ok := got["UPPER.PTOR"]; ok {
		t.Fatalf("uppercase key should be normalized")
	}
}

func TestResolveAddress_CaseInsensitive(t *testing.T) {
	dir := entity.Directory{HiddenServices: map[string]entity.HiddenServiceInfo{
		"lower.ptor": {Relay: "exit", PubKey: "pk"},
	}}
	addr, exit, err := resolveAddress(dir, "LOWER.PTOR", 80)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if exit != "exit" {
		t.Fatalf("unexpected exit: %s", exit)
	}
	if addr != "lower.ptor:80" {
		t.Fatalf("unexpected addr: %s", addr)
	}
}
