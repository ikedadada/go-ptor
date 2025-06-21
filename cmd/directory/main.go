package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
)

type Directory struct {
	Relays         map[string]RelayInfo         `json:"relays"`
	HiddenServices map[string]HiddenServiceInfo `json:"hidden_services"`
}

type RelayInfo struct {
	Endpoint string `json:"endpoint"`
	PubKey   string `json:"pubkey"`
}

type HiddenServiceInfo struct {
	Relay  string `json:"relay"`
	PubKey string `json:"pubkey"`
}

func loadDirectory(path string) (Directory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Directory{}, err
	}
	var d Directory
	if err := json.Unmarshal(b, &d); err != nil {
		return Directory{}, err
	}
	return d, nil
}

func handler(d Directory) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	})
}

func main() {
	data := flag.String("data", "directory.json", "directory json file")
	listen := flag.String("listen", ":8081", "listen address")
	flag.Parse()

	dir, err := loadDirectory(*data)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("directory server listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, handler(dir)))
}
