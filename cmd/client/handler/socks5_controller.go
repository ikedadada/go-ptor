package handler

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// SOCKS5Controller handles SOCKS5 proxy connections
type SOCKS5Controller struct {
	hsRepo    repository.HiddenServiceRepository
	cRepo     repository.CircuitRepository
	cSvc      service.CryptoService
	crSvc     service.CellReaderService
	peSvc     service.PayloadEncodingService
	buildUC   usecase.BuildCircuitUseCase
	connectUC usecase.SendConnectUseCase
	openUC    usecase.OpenStreamUseCase
	closeUC   usecase.CloseStreamUseCase
	sendUC    usecase.SendDataUseCase
	endUC     usecase.HandleEndUseCase
	hops      int
}

// NewSOCKS5Controller creates a new SOCKS5Controller
func NewSOCKS5Controller(
	hsRepo repository.HiddenServiceRepository,
	cRepo repository.CircuitRepository,
	cSvc service.CryptoService,
	crSvc service.CellReaderService,
	peSvc service.PayloadEncodingService,
	buildUC usecase.BuildCircuitUseCase,
	connectUC usecase.SendConnectUseCase,
	openUC usecase.OpenStreamUseCase,
	closeUC usecase.CloseStreamUseCase,
	sendUC usecase.SendDataUseCase,
	endUC usecase.HandleEndUseCase,
	hops int,
) *SOCKS5Controller {
	return &SOCKS5Controller{
		hsRepo:    hsRepo,
		cRepo:     cRepo,
		cSvc:      cSvc,
		crSvc:     crSvc,
		peSvc:     peSvc,
		buildUC:   buildUC,
		connectUC: connectUC,
		openUC:    openUC,
		closeUC:   closeUC,
		sendUC:    sendUC,
		endUC:     endUC,
		hops:      hops,
	}
}

// SOCKS5Request represents a parsed SOCKS5 connection request
type SOCKS5Request struct {
	Host string
	Port int
}

// HandleConnection handles a SOCKS5 connection
func (c *SOCKS5Controller) HandleConnection(conn net.Conn) {
	defer conn.Close()

	// Phase 1: Handle SOCKS5 protocol negotiation
	req, err := c.handleSOCKS5Protocol(conn)
	if err != nil {
		log.Println("SOCKS5 protocol error:", err)
		return
	}

	// Phase 2: Resolve target address and build circuit
	addr, exitID, err := c.resolveAddress(req.Host, req.Port)
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write(vo.SOCKS5HostUnreachResp)
		return
	}

	circuitID, err := c.buildCircuit(exitID)
	if err != nil {
		log.Println("build circuit:", err)
		conn.Write(vo.SOCKS5ErrorResp)
		return
	}

	// Phase 3: Setup stream management and start data relay
	if err := c.setupStreamAndRelay(conn, circuitID, exitID, addr); err != nil {
		log.Println("setup stream and relay:", err)
		return
	}
}

// handleSOCKS5Protocol handles the SOCKS5 protocol handshake and request parsing
func (c *SOCKS5Controller) handleSOCKS5Protocol(conn net.Conn) (*SOCKS5Request, error) {
	var buf [262]byte

	// Step 1: Read authentication methods
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return nil, fmt.Errorf("read SOCKS version: %w", err)
	}
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		return nil, fmt.Errorf("read SOCKS methods: %w", err)
	}
	conn.Write(vo.SOCKS5HandshakeResp)

	// Step 2: Read connection request
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return nil, fmt.Errorf("read SOCKS request: %w", err)
	}
	if buf[1] != vo.SOCKS5CmdConnect {
		return nil, fmt.Errorf("unsupported SOCKS command: %d", buf[1])
	}

	// Step 3: Parse target address
	host, err := c.parseTargetAddress(conn, buf[3])
	if err != nil {
		return nil, fmt.Errorf("parse target address: %w", err)
	}

	// Step 4: Read target port
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return nil, fmt.Errorf("read port: %w", err)
	}
	port := int(buf[0])<<8 | int(buf[1])

	return &SOCKS5Request{Host: host, Port: port}, nil
}

// parseTargetAddress parses the target address from SOCKS5 request
func (c *SOCKS5Controller) parseTargetAddress(conn net.Conn, addrType byte) (string, error) {
	var buf [262]byte

	switch addrType {
	case vo.SOCKS5AddrIPv4:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return "", fmt.Errorf("read IPv4 address: %w", err)
		}
		return net.IP(buf[:4]).String(), nil

	case vo.SOCKS5AddrDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return "", fmt.Errorf("read hostname length: %w", err)
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return "", fmt.Errorf("read hostname: %w", err)
		}
		return string(buf[:l]), nil

	default:
		return "", fmt.Errorf("unsupported address type: %d", addrType)
	}
}

// buildCircuit builds a circuit for the connection
func (c *SOCKS5Controller) buildCircuit(exitID string) (string, error) {
	log.Printf("building circuit hops=%d exitID=%s", c.hops, exitID)
	buildOut, err := c.buildUC.Handle(usecase.BuildCircuitInput{Hops: c.hops, ExitRelayID: exitID})
	if err != nil {
		return "", fmt.Errorf("build circuit: %w", err)
	}
	log.Printf("circuit built successfully cid=%s", buildOut.CircuitID)
	return buildOut.CircuitID, nil
}

// setupStreamAndRelay sets up stream management and handles data relay
func (c *SOCKS5Controller) setupStreamAndRelay(conn net.Conn, circuitID, exitID, addr string) error {
	// Setup stream manager and receive loop
	cid, _ := vo.CircuitIDFrom(circuitID)
	sm := service.NewStreamManagerService()
	go c.recvLoop(cid, sm)

	// Connect to hidden service if needed
	if exitID != "" {
		if _, err := c.connectUC.Handle(usecase.SendConnectInput{CircuitID: circuitID}); err != nil {
			log.Println("connect hidden:", err)
		}
	}

	// Open stream
	stOut, err := c.openUC.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	sid := stOut.StreamID
	sm.Add(uint16(sid), conn)
	defer sm.Remove(uint16(sid))

	// Send BEGIN command
	if err := c.sendBeginCommand(circuitID, int(sid), addr); err != nil {
		return fmt.Errorf("send begin command: %w", err)
	}
	conn.Write(vo.SOCKS5SuccessResp)

	// Start data relay loop
	c.dataRelayLoop(conn, circuitID, int(sid))

	// Cleanup
	if _, err := c.closeUC.Handle(usecase.CloseStreamInput{CircuitID: circuitID, StreamID: sid}); err != nil {
		log.Println("close stream:", err)
	}
	return nil
}

// sendBeginCommand sends the BEGIN command to establish the stream
func (c *SOCKS5Controller) sendBeginCommand(circuitID string, sid int, addr string) error {
	payload, err := c.peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: uint16(sid), Target: addr})
	if err != nil {
		return fmt.Errorf("encode begin: %w", err)
	}
	log.Printf("sending BEGIN command cid=%s sid=%d target=%s", circuitID, sid, addr)
	if _, err := c.sendUC.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: uint16(sid), Data: payload, Cmd: vo.CmdBegin}); err != nil {
		return fmt.Errorf("send begin: %w", err)
	}
	log.Printf("BEGIN command sent successfully cid=%s sid=%d", circuitID, sid)
	return nil
}

// dataRelayLoop handles the data relay between client and circuit
func (c *SOCKS5Controller) dataRelayLoop(conn net.Conn, circuitID string, sid int) {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			log.Printf("sending DATA command cid=%s sid=%d bytes=%d", circuitID, sid, n)
			if _, err2 := c.sendUC.Handle(usecase.SendDataInput{CircuitID: circuitID, StreamID: uint16(sid), Data: buf[:n]}); err2 != nil {
				log.Println("send data:", err2)
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				_, _ = c.endUC.Handle(usecase.HandleEndInput{CircuitID: circuitID, StreamID: uint16(sid)})
			}
			break
		}
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
func (c *SOCKS5Controller) recvLoop(cid vo.CircuitID, sm service.StreamManagerService) {
	cir, err := c.cRepo.Find(cid)
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
		case vo.CmdData:
			c.handleDataCell(cell, cir, sm)
		case vo.CmdEnd:
			if c.handleEndCell(cell, sm) {
				return
			}
		case vo.CmdDestroy:
			sm.CloseAll()
			return
		}
	}
}

// handleDataCell processes incoming data cells and decrypts onion layers
func (c *SOCKS5Controller) handleDataCell(cell *entity.Cell, cir *entity.Circuit, sm service.StreamManagerService) {
	dp, err := c.peSvc.DecodeDataPayload(cell.Payload)
	if err != nil {
		return
	}

	// Decrypt multi-layer onion encryption
	data, err := c.decryptOnionLayers(dp.Data, cir)
	if err != nil {
		log.Printf("onion decryption failed: %v", err)
		return
	}

	// Forward decrypted data to the appropriate stream
	if conn, ok := sm.Get(dp.StreamID); ok {
		conn.Write(data)
	}
}

// decryptOnionLayers decrypts multi-layer onion encryption for response data
func (c *SOCKS5Controller) decryptOnionLayers(data []byte, cir *entity.Circuit) ([]byte, error) {
	hopCount := len(cir.Hops())
	log.Printf("response decrypt multi-layer hops=%d dataLen=%d", hopCount, len(data))

	// Decrypt each layer in reverse circuit order (first hop to exit hop)
	for hop := 0; hop < hopCount; hop++ {
		key := cir.HopKey(hop)
		nonce := cir.HopUpstreamDataNonce(hop)

		log.Printf("response decrypt hop=%d nonce=%x key=%x", hop, nonce, key)
		decrypted, err := c.cSvc.AESOpen(key, nonce, data)
		if err != nil {
			return nil, fmt.Errorf("response decrypt failed hop=%d: %w", hop, err)
		}
		data = decrypted
		log.Printf("response decrypt success hop=%d len=%d", hop, len(data))
	}

	return data, nil
}

// handleEndCell processes stream end commands
func (c *SOCKS5Controller) handleEndCell(cell *entity.Cell, sm service.StreamManagerService) bool {
	sid := uint16(0)
	if len(cell.Payload) > 0 {
		if p, err := c.peSvc.DecodeDataPayload(cell.Payload); err == nil {
			sid = p.StreamID
		}
	}

	if sid == 0 {
		// End all streams
		sm.CloseAll()
		return true
	}

	// End specific stream
	sm.Remove(sid)
	return false
}
