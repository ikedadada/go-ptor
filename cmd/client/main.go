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

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraRepo "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

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
	return d.HiddenServices, nil
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
func resolveAddress(dir entity.Directory, host string, port int) (string, error) {
	if strings.HasSuffix(host, ".ptor") {
		hs, ok := dir.HiddenServices[host]
		if !ok {
			return "", fmt.Errorf("hidden service not found: %s", host)
		}
		rel, ok := dir.Relays[hs.Relay]
		if !ok {
			return "", fmt.Errorf("relay %s not found", hs.Relay)
		}
		return rel.Endpoint, nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		return fmt.Sprintf("[%s]:%d", host, port), nil
	}
	return fmt.Sprintf("%s:%d", host, port), nil
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

	dialer := infraSvc.NewMemDialer()
	cryptoSvc := infraSvc.NewCryptoService()
	builder := useSvc.NewCircuitBuildService(relayRepository, circuitRepository, dialer, cryptoSvc)
	buildUC := usecase.NewBuildCircuitUseCase(builder)

	out, err := buildUC.Handle(usecase.BuildCircuitInput{Hops: *hops})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Circuit built:", out.CircuitID)

	tx := infraSvc.NewMemTransmitter(make(chan string, 10))
	openUC := usecase.NewOpenStreamUsecase(circuitRepository)
	closeUC := usecase.NewCloseStreamUsecase(circuitRepository, tx)

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
			handleSOCKS(conn, dir, out.CircuitID, openUC, closeUC)
			log.Printf("response connection closed %s", conn.RemoteAddr())
		}(c)
	}
}

func handleSOCKS(conn net.Conn, dir entity.Directory, circuitID string, open usecase.OpenStreamUseCase, close usecase.CloseStreamUseCase) {
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

	addr, err := resolveAddress(dir, host, port)
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}

	stOut, err := open.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		log.Println("open stream:", err)
		return
	}
	sid := stOut.StreamID

	target, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("dial target:", err)
		conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()
	conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

	go io.Copy(target, conn)
	io.Copy(conn, target)

	if _, err := close.Handle(usecase.CloseStreamInput{CircuitID: circuitID, StreamID: sid}); err != nil {
		log.Println("close stream:", err)
	}
}
