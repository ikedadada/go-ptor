package main

import (
	"crypto/ed25519"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"ikedadada/go-ptor/internal/domain/value_object"
)

func main() {
	keyPath := flag.String("key", "hidden.pem", "ED25519 private key")
	// Default to all interfaces so Docker containers can bind. Validation
	// below restricts to loopback or unspecified addresses.
	listen := flag.String("listen", ":5000", "relay listen address")
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
	if ip == nil && strings.EqualFold(host, "localhost") {
		ip = net.ParseIP("127.0.0.1")
	}
	if ip == nil || !(ip.IsLoopback() || ip.IsUnspecified()) {
		log.Fatal("hidden service must listen on loopback or unspecified address")
	}

	priv, err := loadEDPriv(*keyPath)
	if err != nil {
		log.Fatal(err)
	}
	addr := value_object.NewHiddenAddr(priv.Public().(ed25519.PublicKey))
	fmt.Println("Hidden address:", addr.String())

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
			defer conn.Close()
			upstream, err := net.Dial("tcp", *httpAddr)
			if err != nil {
				log.Printf("dial http service: %v", err)
				return
			}
			defer upstream.Close()
			go io.Copy(upstream, conn)
			io.Copy(conn, upstream)
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
