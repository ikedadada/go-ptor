package main

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/google/uuid"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraRepo "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

func main() {
	entry := flag.String("entry", "127.0.0.1:5000", "entry relay address")
	hops := flag.Int("hops", 3, "number of hops")
	socks := flag.String("socks", "127.0.0.1:9050", "SOCKS5 listen address")
	flag.Parse()

	// --- repositories & services ---
	relayRepository := infraRepo.NewRelayRepository()
	circuitRepository := infraRepo.NewCircuitRepository()

	host, portStr, err := net.SplitHostPort(*entry)
	if err != nil {
		log.Fatal(err)
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal(err)
	}
	ep, err := value_object.NewEndpoint(host, uint16(p))
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < *hops; i++ {
		rid, err := value_object.NewRelayID(uuid.NewString())
		if err != nil {
			log.Fatal(err)
		}
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatal(err)
		}
		rel := entity.NewRelay(rid, ep, value_object.RSAPubKey{PublicKey: &key.PublicKey})
		rel.SetOnline()
		if err := relayRepository.Save(rel); err != nil {
			log.Fatal(err)
		}
	}

	builder := useSvc.NewCircuitBuildService(relayRepository, circuitRepository)
	buildUC := usecase.NewBuildCircuitUseCase(builder)

	out, err := buildUC.Handle(usecase.BuildCircuitInput{Hops: *hops})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Circuit built:", out.CircuitID)

	tx := &infraSvc.MemTransmitter{Out: make(chan string, 10)}
	openUC := usecase.NewOpenStreamInteractor(circuitRepository)
	closeUC := usecase.NewCloseStreamInteractor(circuitRepository, tx)

	ln, err := net.Listen("tcp", *socks)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("SOCKS5 proxy listening on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleSOCKS(c, out.CircuitID, openUC, closeUC)
	}
}

func handleSOCKS(conn net.Conn, circuitID string, open usecase.OpenStreamUseCase, close usecase.CloseStreamUseCase) {
	defer conn.Close()

	var buf [262]byte
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		return
	}
	conn.Write([]byte{5, 0})

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[1] != 1 {
		return
	}
	var host string
	switch buf[3] {
	case 1:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 3:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	default:
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	var addr string
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		// IPv6 address
		addr = fmt.Sprintf("[%s]:%d", host, port)
	} else {
		// IPv4 address or hostname
		addr = fmt.Sprintf("%s:%d", host, port)
	}

	stOut, err := open.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		return
	}
	sid := stOut.StreamID

	target, err := net.Dial("tcp", addr)
	if err != nil {
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
