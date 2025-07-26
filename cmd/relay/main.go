package main

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"ikedadada/go-ptor/cmd/relay/handler"
	repoimpl "ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func main() {
	listen := flag.String("listen", ":5000", "listen address")
	privPath := flag.String("priv", "", "RSA private key")
	ttl := flag.Duration("ttl", defaultTTL(), "circuit entry TTL")
	flag.Parse()
	var priv vo.PrivateKey
	var err error
	if *privPath == "" {
		rawKey, genErr := rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			log.Fatal(genErr)
		}
		priv = vo.NewRSAPrivKey(rawKey)
	} else {
		priv, err = loadPrivateKey(*privPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	repo := repoimpl.NewConnStateRepository(*ttl)
	cryptoSvc := service.NewCryptoService()
	reader := service.NewCellReaderService()
	cellSender := service.NewCellSenderService()

	// Create individual usecases
	extendUC := usecase.NewHandleExtendUseCase(priv, repo, cryptoSvc, cellSender)
	beginUC := usecase.NewHandleBeginUseCase(repo, cryptoSvc, cellSender)
	dataUC := usecase.NewHandleDataUseCase(repo, cryptoSvc, cellSender)
	endStreamUC := usecase.NewHandleEndStreamUseCase(repo, cellSender)
	destroyUC := usecase.NewHandleDestroyUseCase(repo, cellSender)
	connectUC := usecase.NewHandleConnectUseCase(repo, cryptoSvc, cellSender)

	// Create handler with all usecases
	relayHandler := handler.NewRelayHandler(
		repo,
		reader,
		cellSender,
		extendUC,
		beginUC,
		dataUC,
		endStreamUC,
		destroyUC,
		connectUC,
	)

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
			relayHandler.ServeConn(conn)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

// loadPrivateKey loads a private key from a PEM file and returns it as a PrivateKey value object.
func loadPrivateKey(path string) (vo.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return vo.ParsePrivateKeyFromPEM(b)
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
