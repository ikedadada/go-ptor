package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/handler"
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

	dialer := infraSvc.NewTCPDialer()
	cryptoSvc := infraSvc.NewCryptoService()
	builder := useSvc.NewCircuitBuildService(relayRepository, circuitRepository, dialer, cryptoSvc)
	buildUC := usecase.NewBuildCircuitUseCase(builder)

	out, err := buildUC.Handle(usecase.BuildCircuitInput{Hops: *hops})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Circuit built:", out.CircuitID)

	factory := infraSvc.TCPTransmitterFactory{}
	openUC := usecase.NewOpenStreamUsecase(circuitRepository)
	closeUC := usecase.NewCloseStreamUsecase(circuitRepository, factory)
	sendUC := usecase.NewSendDataUsecase(circuitRepository, factory, cryptoSvc)
	endUC := usecase.NewHandleEndUsecase(circuitRepository)

	h := handler.NewClientHandler(dir, out.CircuitID, openUC, closeUC, sendUC, endUC)
	_, err = h.StartSOCKS(*socks)
	if err != nil {
		log.Fatal(err)
	}
	select {}
}
