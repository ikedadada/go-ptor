package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"ikedadada/go-ptor/internal/domain/entity"
)

func loadDirectory(path string) (entity.Directory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return entity.Directory{}, err
	}
	var d entity.Directory
	if err := json.Unmarshal(b, &d); err != nil {
		return entity.Directory{}, err
	}
	return d, nil
}

func newMux(d entity.Directory) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entity.Directory{Relays: d.Relays, HiddenServices: d.HiddenServices})
		log.Printf("response %s %s %d", r.Method, r.URL.Path, http.StatusOK)
	})
	return mux
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
	log.Fatal(http.ListenAndServe(*listen, newMux(dir)))
}
