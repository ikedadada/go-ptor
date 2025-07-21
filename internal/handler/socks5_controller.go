package handler

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// SOCKS5Controller handles SOCKS5 proxy connections
type SOCKS5Controller struct {
	hsRepo      repoif.HiddenServiceRepository
	circuitRepo repoif.CircuitRepository
	cryptoSvc   useSvc.CryptoService
	crSvc       useSvc.CellReaderService
	buildUC     usecase.BuildCircuitUseCase
	connectUC   usecase.ConnectUseCase
	openUC      usecase.OpenStreamUseCase
	closeUC     usecase.CloseStreamUseCase
	sendUC      usecase.SendDataUseCase
	endUC       usecase.HandleEndUseCase
	hops        int
}

// NewSOCKS5Controller creates a new SOCKS5Controller
func NewSOCKS5Controller(
	hsRepo repoif.HiddenServiceRepository,
	circuitRepo repoif.CircuitRepository,
	cryptoSvc useSvc.CryptoService,
	crSvc useSvc.CellReaderService,
	buildUC usecase.BuildCircuitUseCase,
	connectUC usecase.ConnectUseCase,
	openUC usecase.OpenStreamUseCase,
	closeUC usecase.CloseStreamUseCase,
	sendUC usecase.SendDataUseCase,
	endUC usecase.HandleEndUseCase,
	hops int,
) *SOCKS5Controller {
	return &SOCKS5Controller{
		hsRepo:      hsRepo,
		circuitRepo: circuitRepo,
		cryptoSvc:   cryptoSvc,
		crSvc:       crSvc,
		buildUC:     buildUC,
		connectUC:   connectUC,
		openUC:      openUC,
		closeUC:     closeUC,
		sendUC:      sendUC,
		endUC:       endUC,
		hops:        hops,
	}
}

// HandleConnection handles a SOCKS5 connection
func (c *SOCKS5Controller) HandleConnection(conn net.Conn) {
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

	addr, exitID, err := c.resolveAddress(host, port)
	hidden := exitID != ""
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write(value_object.SOCKS5HostUnreachResp)
		return
	}

	log.Printf("building circuit hops=%d exitID=%s", c.hops, exitID)
	buildOut, err := c.buildUC.Handle(usecase.BuildCircuitInput{Hops: c.hops, ExitRelayID: exitID})
	if err != nil {
		log.Println("build circuit:", err)
		conn.Write(value_object.SOCKS5ErrorResp)
		return
	}
	circuitID := buildOut.CircuitID
	log.Printf("circuit built successfully cid=%s", circuitID)

	cid, _ := value_object.CircuitIDFrom(circuitID)
	sm := useSvc.NewStreamManagerService()
	go c.recvLoop(cid, sm)

	if hidden {
		if _, err := c.connectUC.Handle(usecase.ConnectInput{CircuitID: circuitID}); err != nil {
			log.Println("connect hidden:", err)
		}
	}

	stOut, err := c.openUC.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
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
	if _, err := c.sendUC.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: sid, Data: payload, Cmd: value_object.CmdBegin}); err != nil {
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
			if _, err2 := c.sendUC.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: sid, Data: bufLocal[:n]}); err2 != nil {
				log.Println("send data:", err2)
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				_, _ = c.endUC.Handle(usecase.HandleEndInput{CircuitID: circuitID, StreamID: sid})
			}
			break
		}
	}

	if _, err := c.closeUC.Handle(usecase.CloseStreamInput{CircuitID: circuitID, StreamID: sid}); err != nil {
		log.Println("close stream:", err)
	}
}

// ResolveAddress returns the dial address for the given host and port.
// If host ends with .ptor, it looks up the hidden service in the repository
// and returns the endpoint of the designated exit relay.
func (c *SOCKS5Controller) ResolveAddress(host string, port int) (string, string, error) {
	return c.resolveAddress(host, port)
}

// resolveAddress returns the dial address for the given host and port.
// If host ends with .ptor, it looks up the hidden service in the repository
// and returns the endpoint of the designated exit relay.
func (c *SOCKS5Controller) resolveAddress(host string, port int) (string, string, error) {
	hostLower := strings.ToLower(host)
	exit := ""
	if strings.HasSuffix(hostLower, ".ptor") {
		hs, err := c.hsRepo.FindByAddressString(hostLower)
		if err != nil {
			return "", "", fmt.Errorf("hidden service not found: %s", host)
		}
		exit = hs.RelayID().String()
	}
	if ip := net.ParseIP(hostLower); ip != nil && ip.To4() == nil {
		return fmt.Sprintf("[%s]:%d", hostLower, port), exit, nil
	}
	return fmt.Sprintf("%s:%d", hostLower, port), exit, nil
}

// recvLoop handles incoming data from the circuit
func (c *SOCKS5Controller) recvLoop(cid value_object.CircuitID, sm useSvc.StreamManagerService) {
	cir, err := c.circuitRepo.Find(cid)
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
		_, cell, err := c.crSvc.ReadCell(conn)
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
				decrypted, err := c.cryptoSvc.AESOpen(key, nonce, data)
				if err != nil {
					log.Printf("response decrypt failed hop=%d: %v", hop, err)
					break
				}
				data = decrypted
				log.Printf("response decrypt success hop=%d len=%d", hop, len(data))
			}

			if conn, ok := sm.Get(dp.StreamID); ok {
				conn.Write(data)
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
