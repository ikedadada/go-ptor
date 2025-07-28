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
	buildUC       usecase.BuildCircuitUseCase
	connectUC     usecase.SendConnectUseCase
	openUC        usecase.OpenStreamUseCase
	closeUC       usecase.CloseStreamUseCase
	sendUC        usecase.SendDataUseCase
	endUC         usecase.HandleEndUseCase
	resolveUC     usecase.ResolveTargetAddressUseCase
	receiveCellUC usecase.ReceiveCellUseCase
	decryptCellUC usecase.DecryptCellDataUseCase
	peSvc         service.PayloadEncodingService
	smSvc         service.StreamManagerService
	hops          int
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
	receiveCellUC usecase.ReceiveCellUseCase,
	decryptCellUC usecase.DecryptCellDataUseCase,
	peSvc service.PayloadEncodingService,
	smSvc service.StreamManagerService,
	hops int,
) *SOCKS5Controller {
	return &SOCKS5Controller{
		buildUC:       buildUC,
		connectUC:     connectUC,
		openUC:        openUC,
		closeUC:       closeUC,
		sendUC:        sendUC,
		endUC:         endUC,
		resolveUC:     resolveUC,
		receiveCellUC: receiveCellUC,
		decryptCellUC: decryptCellUC,
		peSvc:         peSvc,
		smSvc:         smSvc,
		hops:          hops,
	}
}

// HandleConnection handles a SOCKS5 connection
func (c *SOCKS5Controller) HandleConnection(conn net.Conn) {
	defer conn.Close()

	// Phase 1: Handle SOCKS5 protocol negotiation
	log.Println("Starting SOCKS5 protocol handling")
	var buf [262]byte

	// Step 1: Read authentication methods
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		log.Printf("read SOCKS version: %v", err)
		return
	}
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		log.Printf("read SOCKS methods: %v", err)
		return
	}
	conn.Write(vo.SOCKS5HandshakeResp)

	// Step 2: Read connection request
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		log.Printf("read SOCKS request: %v", err)
		return
	}
	if buf[1] != vo.SOCKS5CmdConnect {
		log.Printf("unsupported SOCKS command: %d", buf[1])
		return
	}

	// Step 3: Parse target address
	var host string
	switch buf[3] {
	case vo.SOCKS5AddrIPv4:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			log.Printf("read IPv4 address: %v", err)
			return
		}
		host = net.IP(buf[:4]).String()
	case vo.SOCKS5AddrDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			log.Printf("read hostname length: %v", err)
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			log.Printf("read hostname: %v", err)
			return
		}
		host = string(buf[:l])
	default:
		log.Printf("unsupported address type: %d", buf[3])
		return
	}

	// Step 4: Read target port
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		log.Printf("read port: %v", err)
		return
	}
	port := int(buf[0])<<8 | int(buf[1])

	log.Printf("SOCKS5 protocol completed, target: %s:%d", host, port)

	// Phase 2: Resolve target address and build circuit
	resolveOut, err := c.resolveUC.Handle(usecase.ResolveTargetAddressInput{
		Host: host,
		Port: port,
	})
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write(vo.SOCKS5HostUnreachResp)
		return
	}
	addr := resolveOut.DialAddress
	exitRelayID := resolveOut.ExitRelayID

	log.Printf("building circuit hops=%d exitRelayID=%s", c.hops, exitRelayID)
	buildOut, err := c.buildUC.Handle(usecase.BuildCircuitInput{Hops: c.hops, ExitRelayID: exitRelayID})
	if err != nil {
		log.Printf("build circuit: %v", err)
		conn.Write(vo.SOCKS5ErrorResp)
		return
	}
	circuitID := buildOut.CircuitID
	log.Printf("circuit built successfully cid=%s", circuitID)

	// Phase 3: Setup stream management and start data relay
	if err := c.setupStreamAndRelay(conn, circuitID, exitRelayID, addr); err != nil {
		log.Println("setup stream and relay:", err)
		return
	}
}

// setupStreamAndRelay sets up stream management and handles data relay
func (c *SOCKS5Controller) setupStreamAndRelay(conn net.Conn, circuitID, exitRelayID, addr string) error {
	// === 1. Initialize stream manager and start receiving loop ===
	go c.recvLoop(circuitID)

	// === 2. Connect to hidden service if needed ===
	if exitRelayID != "" {
		log.Printf("connecting to hidden service cid=%s", circuitID)
		if _, err := c.connectUC.Handle(usecase.SendConnectInput{CircuitID: circuitID}); err != nil {
			log.Printf("failed to connect to hidden service cid=%s: %v", circuitID, err)
			return fmt.Errorf("connect to hidden service: %w", err)
		}
		log.Printf("hidden service connection established cid=%s", circuitID)
	}

	// === 3. Open stream and register with manager ===
	stOut, err := c.openUC.Handle(usecase.OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	streamID := stOut.StreamID
	c.smSvc.Add(uint16(streamID), conn)
	defer c.smSvc.Remove(uint16(streamID))
	log.Printf("stream opened and registered cid=%s sid=%d", circuitID, streamID)

	// === 4. Send BEGIN command to establish stream connection ===
	payload, err := c.peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{
		StreamID: uint16(streamID),
		Target:   addr,
	})
	if err != nil {
		return fmt.Errorf("encode begin payload: %w", err)
	}

	log.Printf("establishing stream connection cid=%s sid=%d target=%s", circuitID, streamID, addr)
	_, err = c.sendUC.Handle(usecase.SendDataInput{
		CircuitID: circuitID,
		StreamID:  uint16(streamID),
		Data:      payload,
		Cmd:       vo.CmdBegin,
	})
	if err != nil {
		return fmt.Errorf("send begin command: %w", err)
	}
	log.Printf("stream connection established cid=%s sid=%d", circuitID, streamID)

	// === 5. Notify client of successful connection ===
	conn.Write(vo.SOCKS5SuccessResp)

	// === 6. Handle data relay loop ===
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)

		// Send any received data to the circuit
		if n > 0 {
			log.Printf("relaying data cid=%s sid=%d bytes=%d", circuitID, streamID, n)
			_, sendErr := c.sendUC.Handle(usecase.SendDataInput{
				CircuitID: circuitID,
				StreamID:  uint16(streamID),
				Data:      buf[:n],
			})
			if sendErr != nil {
				log.Printf("failed to send data: %v", sendErr)
				break
			}
		}

		// Handle connection errors
		if err != nil {
			if err == io.EOF {
				log.Printf("client connection closed cid=%s sid=%d", circuitID, streamID)
				_, _ = c.endUC.Handle(usecase.HandleEndInput{
					CircuitID: circuitID,
					StreamID:  uint16(streamID),
				})
			} else {
				log.Printf("client connection error cid=%s sid=%d: %v", circuitID, streamID, err)
			}
			break
		}
	}

	// === 7. Cleanup stream resources ===
	if _, err := c.closeUC.Handle(usecase.CloseStreamInput{
		CircuitID: circuitID,
		StreamID:  streamID,
	}); err != nil {
		log.Printf("failed to close stream cid=%s sid=%d: %v", circuitID, streamID, err)
	}

	return nil
}

// recvLoop handles incoming data from the circuit
func (c *SOCKS5Controller) recvLoop(circuitID string) {
	for {
		// Step 1: Receive cell from circuit
		receiveOut, err := c.receiveCellUC.Handle(usecase.ReceiveCellInput{
			CircuitID: circuitID,
		})
		if err != nil {
			log.Println("receive cell:", err)
			c.smSvc.CloseAll()
			return
		}

		if receiveOut.IsEOF {
			log.Println("connection closed")
			c.smSvc.CloseAll()
			return
		}

		// Step 2: Decrypt and process cell data
		decryptOut, err := c.decryptCellUC.Handle(usecase.DecryptCellDataInput{
			Cell:    receiveOut.Cell,
			Circuit: receiveOut.Circuit,
		})
		if err != nil {
			log.Println("decrypt cell:", err)
			c.smSvc.CloseAll()
			return
		}

		if decryptOut.ShouldClose {
			log.Println("circuit destroyed or all streams closed")
			c.smSvc.CloseAll()
			return
		}

		if decryptOut.CellData != nil {
			switch decryptOut.CellData.Command {
			case vo.CmdData:
				// Forward decrypted data to the appropriate stream
				if conn, ok := c.smSvc.Get(decryptOut.CellData.StreamID); ok {
					conn.Write(decryptOut.CellData.Data)
				}
			case vo.CmdEnd:
				if decryptOut.CellData.StreamID == 0 {
					// End all streams
					c.smSvc.CloseAll()
					return
				} else {
					// End specific stream
					c.smSvc.Remove(decryptOut.CellData.StreamID)
				}
			}
		}
	}
}
