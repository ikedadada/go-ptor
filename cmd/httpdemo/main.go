package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	listen := flag.String("listen", ":8080", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s %s", r.Method, r.URL.Path)
		fmt.Fprintln(w, "hello from httpdemo")
		log.Printf("response %s %s %d", r.Method, r.URL.Path, http.StatusOK)
	})

	log.Println("http demo server listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, mux))
}
