package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/handler"
	"ikedadada/go-ptor/internal/infrastructure/http"
	infraRepo "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// Use the existing StreamManager interface from usecase package

func recvLoop(repo repoif.CircuitRepository, crypto useSvc.CryptoService, cid value_object.CircuitID, sm usecase.StreamManager) {
	cir, err := repo.Find(cid)
	if err != nil {
		log.Println("find circuit:", err)
		return
	}
	conn := cir.Conn(0)
	if conn == nil {
		log.Println("no connection for circuit")
		return
	}
	for {
		_, cell, err := handler.ReadCell(conn)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			sm.CloseAll()
			return
		}
		switch cell.Cmd {
		case value_object.CmdData:
			dp, err := value_object.DecodeDataPayload(cell.Payload)
			if err != nil {
				continue
			}
			// Response data uses multi-layer encryption - decrypt layer by layer
			// Start from outermost layer (first hop) to innermost (exit hop)
			data := dp.Data
			hopCount := len(cir.Hops())

			log.Printf("response decrypt multi-layer hops=%d dataLen=%d", hopCount, len(data))

			// Decrypt each layer in reverse circuit order (first hop to exit hop)
			for hop := 0; hop < hopCount; hop++ {
				key := cir.HopKey(hop)
				nonce := cir.HopUpstreamDataNonce(hop)

				log.Printf("response decrypt hop=%d nonce=%x key=%x", hop, nonce, key)
				decrypted, err := crypto.AESOpen(key, nonce, data)
				if err != nil {
					log.Printf("response decrypt failed hop=%d: %v", hop, err)
					break
				}
				data = decrypted
				log.Printf("response decrypt success hop=%d len=%d", hop, len(data))
			}

			if c, ok := sm.Get(dp.StreamID); ok {
				c.Write(data)
			}
		case value_object.CmdEnd:
			sid := uint16(0)
			if len(cell.Payload) > 0 {
				if p, err := value_object.DecodeDataPayload(cell.Payload); err == nil {
					sid = p.StreamID
				}
			}
			if sid == 0 {
				sm.CloseAll()
				return
			}
			sm.Remove(sid)
		case value_object.CmdDestroy:
			sm.CloseAll()
			return
		}
	}
}

// resolveAddress returns the dial address for the given host and port.
// If host ends with .ptor, it looks up the hidden service in the directory
// and returns the endpoint of the designated exit relay.
func resolveAddress(dir entity.Directory, host string, port int) (string, string, error) {
	hostLower := strings.ToLower(host)
	exit := ""
	if strings.HasSuffix(hostLower, ".ptor") {
		hs, ok := dir.HiddenServices[hostLower]
		if !ok {
			return "", "", fmt.Errorf("hidden service not found: %s", host)
		}
		exit = hs.Relay
	}
	if ip := net.ParseIP(hostLower); ip != nil && ip.To4() == nil {
		return fmt.Sprintf("[%s]:%d", hostLower, port), exit, nil
	}
	return fmt.Sprintf("%s:%d", hostLower, port), exit, nil
}

func main() {
	hops := flag.Int("hops", 3, "number of hops")
	socks := flag.String("socks", ":9050", "SOCKS5 listen address")
	dirURL := flag.String("dir", "", "base directory URL")
	flag.Parse()

	// --- repositories & services ---
	if *dirURL == "" {
		log.Fatal("base directory URL required")
	}

	// Initialize HTTP client
	httpClient := http.NewHTTPClient()

	// Initialize RelayRepository with directory data
	relayRepository, err := infraRepo.NewRelayRepository(httpClient, *dirURL)
	if err != nil {
		log.Fatal("initialize relay repository:", err)
	}
	circuitRepository := infraRepo.NewCircuitRepository()

	// Fetch directory for hidden services (relays are already loaded in repository)
	directoryUC := usecase.NewDirectoryServiceUseCase()
	hiddenOut, err := directoryUC.FetchHiddenServices(usecase.DirectoryServiceInput{BaseURL: *dirURL})
	if err != nil {
		log.Fatal("fetch hidden services:", err)
	}
	dir := entity.Directory{HiddenServices: hiddenOut.HiddenServices}

	dialer := infraSvc.NewTCPDialer()
	cryptoSvc := infraSvc.NewCryptoService()
	builder := useSvc.NewCircuitBuildService(relayRepository, circuitRepository, dialer, cryptoSvc)
	buildUC := usecase.NewBuildCircuitUseCase(builder)

	factory := infraSvc.TCPTransmitterFactory{}
	openUC := usecase.NewOpenStreamUsecase(circuitRepository)
	closeUC := usecase.NewCloseStreamUsecase(circuitRepository, factory)
	sendUC := usecase.NewSendDataUsecase(circuitRepository, factory, cryptoSvc)
	connectUC := usecase.NewConnectUseCase(circuitRepository, factory, cryptoSvc)
	endUC := usecase.NewHandleEndUsecase(circuitRepository)

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
			handleSOCKS(conn, dir, *hops, buildUC, connectUC, openUC, closeUC, sendUC, endUC, circuitRepository, cryptoSvc)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

func handleSOCKS(conn net.Conn, dir entity.Directory, hops int, build usecase.BuildCircuitUseCase, connect usecase.ConnectUseCase, open usecase.OpenStreamUseCase, close usecase.CloseStreamUseCase, send usecase.SendDataUseCase, end usecase.HandleEndUseCase, repo repoif.CircuitRepository, crypto useSvc.CryptoService) {
	defer conn.Close()

	var buf [262]byte
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		log.Println("read SOCKS version:", err)
		return
	}
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		log.Println("read SOCKS methods:", err)
		return
	}
	conn.Write(value_object.SOCKS5HandshakeResp)

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		log.Println("read SOCKS request:", err)
		return
	}
	if buf[1] != value_object.SOCKS5CmdConnect {
		log.Println("unsupported SOCKS command:", buf[1])
		return
	}
	var host string
	switch buf[3] {
	case value_object.SOCKS5AddrIPv4:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			log.Println("read IPv4 address:", err)
			return
		}
		host = net.IP(buf[:4]).String()
	case value_object.SOCKS5AddrDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			log.Println("read hostname length:", err)
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			log.Println("read hostname:", err)
			return
		}
		host = string(buf[:l])
	default:
		log.Println("unsupported address type:", buf[3])
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		log.Println("read port:", err)
		return
	}
	port := int(buf[0])<<8 | int(buf[1])

	addr, exitID, err := resolveAddress(dir, host, port)
	hidden := exitID != ""
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write(value_object.SOCKS5HostUnreachResp)
		return
	}

	log.Printf("building circuit hops=%d exitID=%s", hops, exitID)
	buildOut, err := build.Handle(usecase.BuildCircuitInput{Hops: hops, ExitRelayID: exitID})
	if err != nil {
		log.Println("build circuit:", err)
		conn.Write(value_object.SOCKS5ErrorResp)
		return
	}
	circuitID := buildOut.CircuitID
	log.Printf("circuit built successfully cid=%s", circuitID)

	cid, _ := value_object.CircuitIDFrom(circuitID)
	sm := usecase.NewStreamManager()
	go recvLoop(repo, crypto, cid, sm)

	if hidden {
		if _, err := connect.Handle(usecase.ConnectInput{CircuitID: circuitID}); err != nil {
			log.Println("connect hidden:", err)
		}
	}

	stOut, err := open.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		log.Println("open stream:", err)
		return
	}
	sid := stOut.StreamID
	sm.Add(uint16(sid), conn)
	defer sm.Remove(uint16(sid))

	payload, err := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: sid, Target: addr})
	if err != nil {
		log.Println("encode begin:", err)
		return
	}
	log.Printf("sending BEGIN command cid=%s sid=%d target=%s", circuitID, sid, addr)
	if _, err := send.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: sid, Data: payload, Cmd: value_object.CmdBegin}); err != nil {
		log.Println("send begin:", err)
		return
	}
	log.Printf("BEGIN command sent successfully cid=%s sid=%d", circuitID, sid)
	conn.Write(value_object.SOCKS5SuccessResp)

	bufLocal := make([]byte, 4096)
	for {
		n, err := conn.Read(bufLocal)
		if n > 0 {
			log.Printf("sending DATA command cid=%s sid=%d bytes=%d", circuitID, sid, n)
			if _, err2 := send.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: sid, Data: bufLocal[:n]}); err2 != nil {
				log.Println("send data:", err2)
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				_, _ = end.Handle(usecase.HandleEndInput{CircuitID: circuitID, StreamID: sid})
			}
			break
		}
	}

	if _, err := close.Handle(usecase.CloseStreamInput{CircuitID: circuitID, StreamID: sid}); err != nil {
		log.Println("close stream:", err)
	}
}
