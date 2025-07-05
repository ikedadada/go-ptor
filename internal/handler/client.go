package handler

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
)

// streamMap tracks open local connections keyed by stream ID.
type streamMap struct {
	m map[uint16]net.Conn
}

func newStreamMap() *streamMap { return &streamMap{m: make(map[uint16]net.Conn)} }

func (s *streamMap) add(id uint16, c net.Conn)      { s.m[id] = c }
func (s *streamMap) get(id uint16) (net.Conn, bool) { c, ok := s.m[id]; return c, ok }
func (s *streamMap) del(id uint16) {
	if c, ok := s.m[id]; ok {
		c.Close()
		delete(s.m, id)
	}
}

// ClientHandler manages SOCKS connections and circuit cell processing.
type ClientHandler struct {
	Dir       entity.Directory
	CircuitID string

	OpenUC  usecase.OpenStreamUseCase
	CloseUC usecase.CloseStreamUseCase
	SendUC  usecase.SendDataUseCase
	EndUC   usecase.HandleEndUseCase

	streams *streamMap
}

// NewClientHandler creates a handler for an existing circuit.
func NewClientHandler(dir entity.Directory, cid string, open usecase.OpenStreamUseCase, close usecase.CloseStreamUseCase, send usecase.SendDataUseCase, end usecase.HandleEndUseCase) *ClientHandler {
	return &ClientHandler{
		Dir:       dir,
		CircuitID: cid,
		OpenUC:    open,
		CloseUC:   close,
		SendUC:    send,
		EndUC:     end,
		streams:   newStreamMap(),
	}
}

// StartSOCKS launches a SOCKS5 listener on addr and handles connections.
func (h *ClientHandler) StartSOCKS(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		log.Println("SOCKS5 proxy listening on", ln.Addr())
		for {
			c, err := ln.Accept()
			if err != nil {
				log.Println("accept error:", err)
				continue
			}
			log.Printf("request connection from %s", c.RemoteAddr())
			go func(conn net.Conn) {
				h.handleSOCKS(conn)
				log.Printf("response connection closed %s", conn.RemoteAddr())
			}(c)
		}
	}()
	return ln, nil
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

// handleSOCKS implements minimal SOCKS5 CONNECT.
func (h *ClientHandler) handleSOCKS(conn net.Conn) {
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

	addr, err := resolveAddress(h.Dir, host, port)
	if err != nil {
		log.Println("resolve address:", err)
		conn.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}

	stOut, err := h.OpenUC.Handle(usecase.OpenStreamInput{CircuitID: h.CircuitID})
	if err != nil {
		log.Println("open stream:", err)
		return
	}
	sid := stOut.StreamID

	payload, err := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: sid, Target: addr})
	if err != nil {
		log.Println("encode begin:", err)
		return
	}
	if _, err := h.SendUC.Handle(usecase.SendDataInput{CircuitID: h.CircuitID, StreamID: sid, Data: payload, Cmd: value_object.CmdBegin}); err != nil {
		log.Println("send begin:", err)
		return
	}
	conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

	bufLocal := make([]byte, 4096)
	for {
		n, err := conn.Read(bufLocal)
		if n > 0 {
			if _, err2 := h.SendUC.Handle(usecase.SendDataInput{CircuitID: h.CircuitID, StreamID: sid, Data: bufLocal[:n]}); err2 != nil {
				log.Println("send data:", err2)
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				_, _ = h.EndUC.Handle(usecase.HandleEndInput{CircuitID: h.CircuitID, StreamID: sid})
			}
			break
		}
	}

	if _, err := h.CloseUC.Handle(usecase.CloseStreamInput{CircuitID: h.CircuitID, StreamID: sid}); err != nil {
		log.Println("close stream:", err)
	}
}

// RecvLoop is a placeholder for future inbound cell processing.
func (h *ClientHandler) RecvLoop(conn net.Conn) {
	_ = conn // TODO: implement when protocol for inbound cells is defined
}
