package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

func main() {
	listen := flag.String("listen", ":5000", "listen address")
	privPath := flag.String("priv", "", "RSA private key")
	ttl := flag.Duration("ttl", defaultTTL(), "circuit entry TTL")
	flag.Parse()
	var priv *rsa.PrivateKey
	var err error
	if *privPath == "" {
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		priv, err = loadRSAPriv(*privPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	tbl := repoimpl.NewCircuitTableRepository(*ttl)
	cryptoSvc := service.NewCryptoService()
	reader := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, tbl, cryptoSvc, reader)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("relay listening on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		log.Printf("request connection from %s", c.RemoteAddr())
		go func(conn net.Conn) {
			uc.ServeConn(conn)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

// loadRSAPriv loads an RSA private key from a PEM file.
func loadRSAPriv(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, io.ErrUnexpectedEOF
	}
	switch blk.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(blk.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not RSA private key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", blk.Type)
	}
}

// defaultTTL returns the TTL for circuit entries derived from the
// PTOR_TTL_SECONDS environment variable or 1 minute if unset/invalid.
func defaultTTL() time.Duration {
	ttl := time.Minute
	if v := os.Getenv("PTOR_TTL_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			ttl = time.Duration(secs) * time.Second
		}
	}
	return ttl
}
