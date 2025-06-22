package main

import (
	"crypto/ed25519"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"ikedadada/go-ptor/internal/domain/value_object"
)

func main() {
	keyPath := flag.String("key", "hidden.pem", "ED25519 private key")
	listen := flag.String("listen", "127.0.0.1:5000", "relay listen address (localhost only)")
	httpAddr := flag.String("http", "127.0.0.1:8080", "HTTP service address")
	flag.Parse()

	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		log.Fatalf("invalid listen address: %v", err)
	}
	if host == "" {
		host = "0.0.0.0"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		if strings.EqualFold(host, "localhost") {
			ip = net.ParseIP("127.0.0.1")
		}
	}
	if ip == nil || !ip.IsLoopback() {
		log.Fatal("hidden service must listen on localhost only")
	}

	priv, err := loadEDPriv(*keyPath)
	if err != nil {
		log.Fatal(err)
	}
	addr := value_object.NewHiddenAddr(priv.Public().(ed25519.PublicKey))
	fmt.Println("Hidden address:", addr.String())

	go http.ListenAndServe(*httpAddr, demoMux())

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("accepting relay connections on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		log.Printf("request connection from %s", c.RemoteAddr())
		go func(conn net.Conn) {
			io.Copy(conn, conn)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

func loadEDPriv(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("no PEM data")
	}
	return ed25519.PrivateKey(blk.Bytes), nil
}

func demoMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from hidden service"))
	})
	return mux
}
