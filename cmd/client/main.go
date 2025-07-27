package main

import (
	"flag"
	"log"
	"net"

	"ikedadada/go-ptor/cmd/client/handler"
	"ikedadada/go-ptor/cmd/client/infrastructure/http"
	infraRepo "ikedadada/go-ptor/cmd/client/infrastructure/repository"
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/service"
)

func main() {
	hops := flag.Int("hops", 3, "number of hops")
	socks := flag.String("socks", ":9050", "SOCKS5 listen address")
	dirURL := flag.String("dir", "", "base directory URL")
	flag.Parse()

	if *dirURL == "" {
		log.Fatal("base directory URL required")
	}

	// Initialize HTTP client
	httpClient := http.NewHTTPClient()

	// Initialize repositories
	rRepo, err := infraRepo.NewRelayRepository(httpClient, *dirURL)
	if err != nil {
		log.Fatal("initialize relay repository:", err)
	}

	hsRepo, err := infraRepo.NewHiddenServiceRepository(httpClient, *dirURL)
	if err != nil {
		log.Fatal("initialize hidden service repository:", err)
	}

	cRepo := infraRepo.NewCircuitRepository()

	// Initialize services and use cases
	cbSvc := service.NewTCPCircuitBuildService()
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	peSvc := service.NewPayloadEncodingService()
	buildUC := usecase.NewBuildCircuitUseCase(rRepo, cRepo, cbSvc, cSvc, peSvc)

	factory := service.NewTCPMessagingServiceFactory(peSvc)
	openUC := usecase.NewOpenStreamUseCase(cRepo)
	closeUC := usecase.NewCloseStreamUseCase(cRepo, factory)
	sendUC := usecase.NewSendDataUseCase(cRepo, factory, cSvc, peSvc)
	connectUC := usecase.NewSendConnectUseCase(cRepo, factory, cSvc, peSvc)
	endUC := usecase.NewHandleEndUseCase(cRepo)

	// Create SOCKS5 controller
	socks5Controller := handler.NewSOCKS5Controller(
		hsRepo,
		cRepo,
		cSvc,
		crSvc,
		peSvc,
		buildUC,
		connectUC,
		openUC,
		closeUC,
		sendUC,
		endUC,
		*hops,
	)

	ln, err := net.Listen("tcp", *socks)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("SOCKS5 proxy listening on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		log.Printf("request connection from %s", c.RemoteAddr())
		go func(conn net.Conn) {
			socks5Controller.HandleConnection(conn)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}
