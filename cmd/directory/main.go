package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
)

// RelayInfo contains metadata for a relay node published by the directory.
type relayDTO struct {
	ID       string `json:"id"` // Unique identifier for the relay
	Endpoint string `json:"endpoint"`
	PubKey   string `json:"pubkey"`
}

// HiddenServiceInfo maps a hidden service address to its relay and public key.
type hiddenServiceDTO struct {
	Address string `json:"address"`
	Relay   string `json:"relay"`
	PubKey  string `json:"pubkey"`
}

type directoryStorage struct {
	relays         []relayDTO
	hiddenServices []hiddenServiceDTO
}

func loadDirectory[T any](path string) (T, error) {
	var d T
	b, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return d, err
	}
	return d, nil
}

func newMux(d directoryStorage) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/relays", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d.relays)
		log.Printf("response %s %s %d", r.Method, r.URL.Path, http.StatusOK)
	})
	mux.HandleFunc("/hidden_services", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d.hiddenServices)
		log.Printf("response %s %s %d", r.Method, r.URL.Path, http.StatusOK)
	})
	return mux
}

func main() {
	listen := flag.String("listen", ":8081", "listen address")
	flag.Parse()

	relays, err := loadDirectory[[]relayDTO]("relays.json")
	if err != nil {
		log.Fatal(err)
	}
	hiddenServices, err := loadDirectory[[]hiddenServiceDTO]("hidden_services.json")
	if err != nil {
		log.Fatal(err)
	}

	storage := directoryStorage{
		relays:         relays,
		hiddenServices: hiddenServices,
	}

	log.Println("directory server listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, newMux(storage)))
}
