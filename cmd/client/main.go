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
	relayRepository, err := infraRepo.NewRelayRepository(httpClient, *dirURL)
	if err != nil {
		log.Fatal("initialize relay repository:", err)
	}

	hiddenServiceRepository, err := infraRepo.NewHiddenServiceRepository(httpClient, *dirURL)
	if err != nil {
		log.Fatal("initialize hidden service repository:", err)
	}

	circuitRepository := infraRepo.NewCircuitRepository()

	// Initialize services and use cases
	dialer := service.NewTCPCircuitBuildService()
	cryptoSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	buildUC := usecase.NewBuildCircuitUseCase(relayRepository, circuitRepository, dialer, cryptoSvc)

	factory := service.TCPMessagingServiceFactory{}
	openUC := usecase.NewOpenStreamUsecase(circuitRepository)
	closeUC := usecase.NewCloseStreamUsecase(circuitRepository, factory)
	sendUC := usecase.NewSendDataUsecase(circuitRepository, factory, cryptoSvc)
	connectUC := usecase.NewConnectUseCase(circuitRepository, factory, cryptoSvc)
	endUC := usecase.NewHandleEndUsecase(circuitRepository)

	// Create SOCKS5 controller
	socks5Controller := handler.NewSOCKS5Controller(
		hiddenServiceRepository,
		circuitRepository,
		cryptoSvc,
		crSvc,
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
