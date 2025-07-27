package handler

import (
	"fmt"
	"io"
	"log"
	"net"

	"ikedadada/go-ptor/cmd/client/usecase"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// SOCKS5Controller handles SOCKS5 proxy connections
type SOCKS5Controller struct {
	buildUC   usecase.BuildCircuitUseCase
	connectUC usecase.SendConnectUseCase
	openUC    usecase.OpenStreamUseCase
	closeUC   usecase.CloseStreamUseCase
	sendUC    usecase.SendDataUseCase
	endUC     usecase.HandleEndUseCase
	resolveUC usecase.ResolveTargetAddressUseCase
	receiveUC usecase.ReceiveAndDecryptDataUseCase
	peSvc     service.PayloadEncodingService
	hops      int
}

// NewSOCKS5Controller creates a new SOCKS5Controller
func NewSOCKS5Controller(
	buildUC usecase.BuildCircuitUseCase,
	connectUC usecase.SendConnectUseCase,
	openUC usecase.OpenStreamUseCase,
	closeUC usecase.CloseStreamUseCase,
	sendUC usecase.SendDataUseCase,
	endUC usecase.HandleEndUseCase,
	resolveUC usecase.ResolveTargetAddressUseCase,
	receiveUC usecase.ReceiveAndDecryptDataUseCase,
	peSvc service.PayloadEncodingService,
	hops int,
) *SOCKS5Controller {
	return &SOCKS5Controller{
		buildUC:   buildUC,
		connectUC: connectUC,
		openUC:    openUC,
		closeUC:   closeUC,
		sendUC:    sendUC,
		endUC:     endUC,
		resolveUC: resolveUC,
		receiveUC: receiveUC,
		peSvc:     peSvc,
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
	log.Println("Starting SOCKS5 protocol handling")
	req, err := c.handleSOCKS5Protocol(conn)
	if err != nil {
		log.Println("SOCKS5 protocol error:", err)
		return
	}
	log.Printf("SOCKS5 protocol completed, target: %s:%d", req.Host, req.Port)

	// Phase 2: Resolve target address and build circuit
	resolveOut, err := c.resolveUC.Handle(usecase.ResolveTargetAddressInput{
		Host: req.Host,
		Port: req.Port,
	})
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write(vo.SOCKS5HostUnreachResp)
		return
	}
	addr := resolveOut.DialAddress
	exitID := resolveOut.ExitRelayID

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
	sm := service.NewStreamManagerService()
	go c.recvLoop(circuitID, sm)

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
	resolveOut, err := c.resolveUC.Handle(usecase.ResolveTargetAddressInput{
		Host: host,
		Port: port,
	})
	if err != nil {
		return "", "", err
	}
	return resolveOut.DialAddress, resolveOut.ExitRelayID, nil
}

// recvLoop handles incoming data from the circuit
func (c *SOCKS5Controller) recvLoop(circuitID string, sm service.StreamManagerService) {
	for {
		recvOut, err := c.receiveUC.Handle(usecase.ReceiveAndDecryptDataInput{
			CircuitID: circuitID,
		})
		if err != nil {
			log.Println("receive data:", err)
			sm.CloseAll()
			return
		}

		if recvOut.IsEOF {
			log.Println("connection closed")
			sm.CloseAll()
			return
		}

		if recvOut.ShouldClose {
			log.Println("circuit destroyed or all streams closed")
			sm.CloseAll()
			return
		}

		if recvOut.CellData != nil {
			switch recvOut.CellData.Command {
			case vo.CmdData:
				// Forward decrypted data to the appropriate stream
				if conn, ok := sm.Get(recvOut.CellData.StreamID); ok {
					conn.Write(recvOut.CellData.Data)
				}
			case vo.CmdEnd:
				if recvOut.CellData.StreamID == 0 {
					// End all streams
					sm.CloseAll()
					return
				} else {
					// End specific stream
					sm.Remove(recvOut.CellData.StreamID)
				}
			}
		}
	}
}
