package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/handler"
	infraRepo "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

type streamMap struct {
	mu sync.Mutex
	m  map[uint16]net.Conn
}

func newStreamMap() *streamMap { return &streamMap{m: make(map[uint16]net.Conn)} }

func (s *streamMap) add(id uint16, c net.Conn) {
	s.mu.Lock()
	s.m[id] = c
	s.mu.Unlock()
}

func (s *streamMap) get(id uint16) (net.Conn, bool) {
	s.mu.Lock()
	c, ok := s.m[id]
	s.mu.Unlock()
	return c, ok
}

func (s *streamMap) remove(id uint16) {
	s.mu.Lock()
	if c, ok := s.m[id]; ok {
		c.Close()
		delete(s.m, id)
	}
	s.mu.Unlock()
}

func (s *streamMap) closeAll() {
	s.mu.Lock()
	for id, c := range s.m {
		if c != nil {
			c.Close()
		}
		delete(s.m, id)
	}
	s.mu.Unlock()
}

func recvLoop(repo repoif.CircuitRepository, crypto useSvc.CryptoService, cid value_object.CircuitID, sm *streamMap) {
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
	keys := make([][32]byte, len(cir.Hops()))
	nonces := make([][12]byte, len(cir.Hops()))
	for {
		_, cell, err := handler.ReadCell(conn)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			sm.closeAll()
			return
		}
		switch cell.Cmd {
		case value_object.CmdData:
			dp, err := value_object.DecodeDataPayload(cell.Payload)
			if err != nil {
				continue
			}
			// Use DATA nonces for decryption
			for i := range cir.Hops() {
				keys[i] = cir.HopKey(i)
				nonces[i] = cir.HopDataNonce(i)
			}
			plain, err := crypto.AESMultiOpen(keys, nonces, dp.Data)
			if err != nil {
				continue
			}
			if c, ok := sm.get(dp.StreamID); ok {
				c.Write(plain)
			}
		case value_object.CmdEnd:
			sid := uint16(0)
			if len(cell.Payload) > 0 {
				if p, err := value_object.DecodeDataPayload(cell.Payload); err == nil {
					sid = p.StreamID
				}
			}
			if sid == 0 {
				sm.closeAll()
				return
			}
			sm.remove(sid)
		case value_object.CmdDestroy:
			sm.closeAll()
			return
		}
	}
}

func fetchRelays(base string) (map[string]entity.RelayInfo, error) {
	url := strings.TrimRight(base, "/") + "/relays.json"
	log.Printf("request GET %s", url)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("response GET %s status=%s", url, res.Status)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", res.Status)
	}
	var d entity.Directory
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return nil, err
	}
	return d.Relays, nil
}

func fetchHidden(base string) (map[string]entity.HiddenServiceInfo, error) {
	url := strings.TrimRight(base, "/") + "/hidden.json"
	log.Printf("request GET %s", url)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Printf("response GET %s status=%s", url, res.Status)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", res.Status)
	}
	var d entity.Directory
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return nil, err
	}
	m := make(map[string]entity.HiddenServiceInfo, len(d.HiddenServices))
	for k, v := range d.HiddenServices {
		m[strings.ToLower(k)] = v
	}
	return m, nil
}

func fetchDirectory(base string) (entity.Directory, error) {
	relays, err := fetchRelays(base)
	if err != nil {
		return entity.Directory{}, err
	}
	hidden, err := fetchHidden(base)
	if err != nil {
		return entity.Directory{}, err
	}
	return entity.Directory{Relays: relays, HiddenServices: hidden}, nil
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
	relayRepository := infraRepo.NewRelayRepository()
	circuitRepository := infraRepo.NewCircuitRepository()

	if *dirURL == "" {
		log.Fatal("base directory URL required")
	}

	dir, err := fetchDirectory(*dirURL)
	if err != nil {
		log.Fatal(err)
	}
	for id, info := range dir.Relays {
		rid, err := value_object.NewRelayID(id)
		if err != nil {
			log.Printf("invalid relay id %q: %v", id, err)
			continue
		}
		host, portStr, err := net.SplitHostPort(info.Endpoint)
		if err != nil {
			log.Printf("parse endpoint %q: %v", info.Endpoint, err)
			continue
		}
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("parse port %q: %v", portStr, err)
			continue
		}
		ep, err := value_object.NewEndpoint(host, uint16(p))
		if err != nil {
			log.Printf("new endpoint: %v", err)
			continue
		}
		pk, err := value_object.RSAPubKeyFromPEM([]byte(info.PubKey))
		if err != nil {
			log.Printf("parse pubkey for %s: %v", id, err)
			continue
		}
		rel := entity.NewRelay(rid, ep, pk)
		rel.SetOnline()
		if err := relayRepository.Save(rel); err != nil {
			log.Printf("save relay %s: %v", id, err)
		}
	}

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
	conn.Write([]byte{5, 0})

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		log.Println("read SOCKS request:", err)
		return
	}
	if buf[1] != 1 {
		log.Println("unsupported SOCKS command:", buf[1])
		return
	}
	var host string
	switch buf[3] {
	case 1:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			log.Println("read IPv4 address:", err)
			return
		}
		host = net.IP(buf[:4]).String()
	case 3:
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
		conn.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}

	log.Printf("building circuit hops=%d exitID=%s", hops, exitID)
	buildOut, err := build.Handle(usecase.BuildCircuitInput{Hops: hops, ExitRelayID: exitID})
	if err != nil {
		log.Println("build circuit:", err)
		conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	circuitID := buildOut.CircuitID
	log.Printf("circuit built successfully cid=%s", circuitID)

	cid, _ := value_object.CircuitIDFrom(circuitID)
	sm := newStreamMap()
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
	sm.add(uint16(sid), conn)
	defer sm.remove(uint16(sid))

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
	conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

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
