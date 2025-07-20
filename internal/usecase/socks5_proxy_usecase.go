package usecase

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
)

// SOCKS5ProxyUseCase handles SOCKS5 protocol interactions
type SOCKS5ProxyUseCase interface {
	HandleConnection(input SOCKS5ConnectionInput) (SOCKS5ConnectionOutput, error)
}

type SOCKS5ConnectionInput struct {
	Conn      net.Conn
	Directory entity.Directory
	Hops      int
	
	// Dependencies for circuit management
	BuildUC   BuildCircuitUseCase
	ConnectUC ConnectUseCase
	OpenUC    OpenStreamUseCase
	CloseUC   CloseStreamUseCase
	SendUC    SendDataUseCase
	EndUC     HandleEndUseCase
}

type SOCKS5ConnectionOutput struct {
	Success   bool
	CircuitID string
	StreamID  uint16
	Target    string
}

type socks5ProxyUseCaseImpl struct{}

func NewSOCKS5ProxyUseCase() SOCKS5ProxyUseCase {
	return &socks5ProxyUseCaseImpl{}
}

func (uc *socks5ProxyUseCaseImpl) HandleConnection(input SOCKS5ConnectionInput) (SOCKS5ConnectionOutput, error) {
	conn := input.Conn
	defer conn.Close()

	// SOCKS5 handshake
	if err := uc.performHandshake(conn); err != nil {
		return SOCKS5ConnectionOutput{}, fmt.Errorf("handshake failed: %w", err)
	}

	// Parse SOCKS5 request
	host, port, err := uc.parseSOCKSRequest(conn)
	if err != nil {
		uc.sendSOCKSError(conn, 4) // Host unreachable
		return SOCKS5ConnectionOutput{}, fmt.Errorf("parse request failed: %w", err)
	}

	// Resolve address and determine if it's a hidden service
	addr, exitID, err := ResolveAddress(input.Directory, host, port)
	if err != nil {
		uc.sendSOCKSError(conn, 4) // Host unreachable
		return SOCKS5ConnectionOutput{}, fmt.Errorf("resolve address failed: %w", err)
	}

	hidden := exitID != ""
	log.Printf("building circuit hops=%d exitID=%s", input.Hops, exitID)

	// Build circuit
	buildOut, err := input.BuildUC.Handle(BuildCircuitInput{
		Hops:        input.Hops,
		ExitRelayID: exitID,
	})
	if err != nil {
		uc.sendSOCKSError(conn, 1) // General failure
		return SOCKS5ConnectionOutput{}, fmt.Errorf("build circuit failed: %w", err)
	}

	circuitID := buildOut.CircuitID
	log.Printf("circuit built successfully cid=%s", circuitID)

	// Handle hidden service connection if needed
	if hidden {
		if _, err := input.ConnectUC.Handle(ConnectInput{CircuitID: circuitID}); err != nil {
			log.Println("connect hidden:", err)
		}
	}

	// Open stream
	stOut, err := input.OpenUC.Handle(OpenStreamInput{CircuitID: circuitID})
	if err != nil {
		uc.sendSOCKSError(conn, 1) // General failure
		return SOCKS5ConnectionOutput{}, fmt.Errorf("open stream failed: %w", err)
	}

	sid := stOut.StreamID

	// Send BEGIN command
	payload, err := value_object.EncodeBeginPayload(&value_object.BeginPayload{
		StreamID: sid,
		Target:   addr,
	})
	if err != nil {
		return SOCKS5ConnectionOutput{}, fmt.Errorf("encode begin failed: %w", err)
	}

	log.Printf("sending BEGIN command cid=%s sid=%d target=%s", circuitID, sid, addr)
	if _, err := input.SendUC.Handle(SendDataInput{
		CircuitID: circuitID,
		StreamID:  sid,
		Data:      payload,
		Cmd:       value_object.CmdBegin,
	}); err != nil {
		return SOCKS5ConnectionOutput{}, fmt.Errorf("send begin failed: %w", err)
	}

	log.Printf("BEGIN command sent successfully cid=%s sid=%d", circuitID, sid)

	// Send success response
	if err := uc.sendSOCKSSuccess(conn); err != nil {
		return SOCKS5ConnectionOutput{}, fmt.Errorf("send success failed: %w", err)
	}

	return SOCKS5ConnectionOutput{
		Success:   true,
		CircuitID: circuitID,
		StreamID:  sid,
		Target:    addr,
	}, nil
}

func (uc *socks5ProxyUseCaseImpl) performHandshake(conn net.Conn) error {
	var buf [262]byte
	
	// Read version and method count
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	
	// Read methods
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		return fmt.Errorf("read methods: %w", err)
	}
	
	// Send response (no authentication required)
	if _, err := conn.Write([]byte{5, 0}); err != nil {
		return fmt.Errorf("send handshake response: %w", err)
	}
	
	return nil
}

func (uc *socks5ProxyUseCaseImpl) parseSOCKSRequest(conn net.Conn) (string, int, error) {
	var buf [262]byte
	
	// Read request header
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return "", 0, fmt.Errorf("read request header: %w", err)
	}
	
	if buf[1] != 1 { // Only support CONNECT command
		return "", 0, fmt.Errorf("unsupported command: %d", buf[1])
	}
	
	var host string
	switch buf[3] { // Address type
	case 1: // IPv4
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return "", 0, fmt.Errorf("read IPv4: %w", err)
		}
		host = net.IP(buf[:4]).String()
	case 3: // Domain name
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return "", 0, fmt.Errorf("read hostname length: %w", err)
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return "", 0, fmt.Errorf("read hostname: %w", err)
		}
		host = string(buf[:l])
	default:
		return "", 0, fmt.Errorf("unsupported address type: %d", buf[3])
	}
	
	// Read port
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return "", 0, fmt.Errorf("read port: %w", err)
	}
	port := int(buf[0])<<8 | int(buf[1])
	
	return host, port, nil
}

// ResolveAddress resolves the address and returns the dial address and exit relay ID
func ResolveAddress(dir entity.Directory, host string, port int) (string, string, error) {
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

func (uc *socks5ProxyUseCaseImpl) sendSOCKSError(conn net.Conn, code byte) {
	// SOCKS5 error response: VER(1) REP(1) RSV(1) ATYP(1) BND.ADDR(4) BND.PORT(2)
	response := []byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0}
	conn.Write(response)
}

func (uc *socks5ProxyUseCaseImpl) sendSOCKSSuccess(conn net.Conn) error {
	// SOCKS5 success response: VER(1) REP(1) RSV(1) ATYP(1) BND.ADDR(4) BND.PORT(2)
	response := []byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	_, err := conn.Write(response)
	return err
}