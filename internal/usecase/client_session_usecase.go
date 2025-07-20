package usecase

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// ClientSessionUseCase manages client proxy sessions and data flow
type ClientSessionUseCase interface {
	StartDataRelay(input DataRelayInput) error
	HandleDataTransfer(input DataTransferInput) error
}

type DataRelayInput struct {
	CircuitID        value_object.CircuitID
	StreamID         uint16
	ClientConnection net.Conn
}

type DataTransferInput struct {
	CircuitID        value_object.CircuitID
	ClientConnection net.Conn
	StreamMap        StreamManager
}

// StreamManager provides thread-safe stream management
type StreamManager interface {
	Add(id uint16, conn net.Conn)
	Get(id uint16) (net.Conn, bool)
	Remove(id uint16)
	CloseAll()
}

type clientSessionUseCaseImpl struct {
	cr  repository.CircuitRepository
	cs  service.CryptoService
	crs service.CellReaderService
	suc SendDataUseCase
	euc HandleEndUseCase
}

func NewClientSessionUseCase(
	cr repository.CircuitRepository,
	cs service.CryptoService,
	crs service.CellReaderService,
	suc SendDataUseCase,
	euc HandleEndUseCase,
) ClientSessionUseCase {
	return &clientSessionUseCaseImpl{
		cr:  cr,
		cs:  cs,
		crs: crs,
		suc: suc,
		euc: euc,
	}
}

func (uc *clientSessionUseCaseImpl) StartDataRelay(input DataRelayInput) error {
	// Start receive loop in goroutine
	go uc.receiveLoop(input.CircuitID, &streamMapImpl{
		m: make(map[uint16]net.Conn),
	})

	// Handle outgoing data from client
	return uc.handleOutgoingData(input)
}

func (uc *clientSessionUseCaseImpl) HandleDataTransfer(input DataTransferInput) error {
	return uc.receiveLoop(input.CircuitID, input.StreamMap)
}

func (uc *clientSessionUseCaseImpl) receiveLoop(cid value_object.CircuitID, sm StreamManager) error {
	cir, err := uc.cr.Find(cid)
	if err != nil {
		return fmt.Errorf("find circuit: %w", err)
	}

	conn := cir.Conn(0)
	if conn == nil {
		return fmt.Errorf("no connection for circuit")
	}

	defer sm.CloseAll() // Ensure cleanup on exit

	for {
		_, cell, err := uc.crs.ReadCell(conn)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return err
		}

		switch cell.Cmd {
		case value_object.CmdData:
			if err := uc.handleDataCell(cell, cir, sm); err != nil {
				log.Println("handle data cell:", err)
				continue
			}
		case value_object.CmdEnd:
			if err := uc.handleEndCell(cell, sm); err != nil {
				log.Println("handle end cell:", err)
			}
		case value_object.CmdDestroy:
			return nil
		}
	}
}

func (uc *clientSessionUseCaseImpl) handleDataCell(cell *value_object.Cell, cir *entity.Circuit, sm StreamManager) error {
	dp, err := value_object.DecodeDataPayload(cell.Payload)
	if err != nil {
		return fmt.Errorf("decode data payload: %w", err)
	}

	// Decrypt response data using multi-layer decryption
	decrypted, err := uc.decryptResponseData(dp.Data, cir, uc.cs)
	if err != nil {
		return fmt.Errorf("decrypt response data: %w", err)
	}

	// Forward decrypted data to client connection
	if c, ok := sm.Get(dp.StreamID); ok {
		if _, err := c.Write(decrypted); err != nil {
			log.Printf("write to client stream %d: %v", dp.StreamID, err)
		}
	}

	return nil
}

func (uc *clientSessionUseCaseImpl) handleEndCell(cell *value_object.Cell, sm StreamManager) error {
	sid := uint16(0)
	if len(cell.Payload) > 0 {
		if p, err := value_object.DecodeDataPayload(cell.Payload); err == nil {
			sid = p.StreamID
		}
	}

	if sid == 0 {
		sm.CloseAll()
		return nil
	}

	sm.Remove(sid)
	return nil
}

func (uc *clientSessionUseCaseImpl) decryptResponseData(data []byte, cir *entity.Circuit, crypto service.CryptoService) ([]byte, error) {
	hopCount := len(cir.Hops())
	log.Printf("response decrypt multi-layer hops=%d dataLen=%d", hopCount, len(data))

	result := data
	// Decrypt each layer in reverse circuit order (first hop to exit hop)
	for hop := 0; hop < hopCount; hop++ {
		key := cir.HopKey(hop)
		nonce := cir.HopUpstreamDataNonce(hop)

		log.Printf("response decrypt hop=%d nonce=%x key=%x", hop, nonce, key)
		decrypted, err := crypto.AESOpen(key, nonce, result)
		if err != nil {
			return nil, fmt.Errorf("decrypt hop %d failed: %w", hop, err)
		}
		result = decrypted
		log.Printf("response decrypt success hop=%d len=%d", hop, len(result))
	}

	return result, nil
}

func (uc *clientSessionUseCaseImpl) handleOutgoingData(input DataRelayInput) error {
	buffer := make([]byte, 4096)
	for {
		n, err := input.ClientConnection.Read(buffer)
		if n > 0 {
			log.Printf("sending DATA command cid=%s sid=%d bytes=%d", input.CircuitID, input.StreamID, n)
			if _, err2 := uc.suc.Handle(SendDataInput{
				CircuitID: input.CircuitID.String(),
				StreamID:  input.StreamID,
				Data:      buffer[:n],
			}); err2 != nil {
				return fmt.Errorf("send data: %w", err2)
			}
		}
		if err != nil {
			if err == io.EOF {
				_, _ = uc.euc.Handle(HandleEndInput{
					CircuitID: input.CircuitID.String(),
					StreamID:  input.StreamID,
				})
			}
			return err
		}
	}
}

// streamMapImpl provides a concrete implementation of StreamManager
type streamMapImpl struct {
	mu sync.Mutex
	m  map[uint16]net.Conn
}

func NewStreamManager() StreamManager {
	return &streamMapImpl{m: make(map[uint16]net.Conn)}
}

func (s *streamMapImpl) Add(id uint16, conn net.Conn) {
	s.mu.Lock()
	s.m[id] = conn
	s.mu.Unlock()
}

func (s *streamMapImpl) Get(id uint16) (net.Conn, bool) {
	s.mu.Lock()
	c, ok := s.m[id]
	s.mu.Unlock()
	return c, ok
}

func (s *streamMapImpl) Remove(id uint16) {
	s.mu.Lock()
	if c, ok := s.m[id]; ok {
		c.Close()
		delete(s.m, id)
	}
	s.mu.Unlock()
}

func (s *streamMapImpl) CloseAll() {
	s.mu.Lock()
	for id, c := range s.m {
		if c != nil {
			c.Close()
		}
		delete(s.m, id)
	}
	s.mu.Unlock()
}
