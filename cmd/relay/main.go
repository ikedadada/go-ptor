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
	"time"

	"ikedadada/go-ptor/internal/handler"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	"ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
)

func main() {
	listen := flag.String("listen", ":5000", "listen address")
	privPath := flag.String("priv", "", "RSA private key")
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
	tbl := repoimpl.NewCircuitTableRepository(time.Minute)
	cryptoSvc := service.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, tbl, cryptoSvc)

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
			handleConn(conn, uc)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

func handleConn(c net.Conn, uc usecase.RelayUseCase) {
	defer c.Close()
	for {
		cid, cell, err := handler.ReadCell(c)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return
		}
		log.Printf("cell cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
		if err := uc.Handle(c, cid, cell); err != nil {
			log.Println("handle:", err)
		}
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
