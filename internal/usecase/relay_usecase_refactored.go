package usecase

import (
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/service"
	"ikedadada/go-ptor/internal/domain/value_object"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// RelayUseCaseRefactored demonstrates DDD refactoring using domain services
// This version separates complex domain logic into appropriate domain services
type RelayUseCaseRefactored interface {
	Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error
	ServeConn(c net.Conn)
}

type relayUsecaseRefactoredImpl struct {
	priv                 *rsa.PrivateKey
	repo                 repoif.CircuitTableRepository
	reader               useSvc.CellReaderService
	cryptographyService  service.CircuitCryptographyService
	relayBehaviorService service.RelayBehaviorService
	cellRoutingService   service.CellRoutingService
}

// NewRelayUseCaseRefactored creates a new refactored relay use case with domain services
func NewRelayUseCaseRefactored(
	priv *rsa.PrivateKey,
	repo repoif.CircuitTableRepository,
	reader useSvc.CellReaderService,
	cryptographyService service.CircuitCryptographyService,
	relayBehaviorService service.RelayBehaviorService,
	cellRoutingService service.CellRoutingService,
) RelayUseCaseRefactored {
	return &relayUsecaseRefactoredImpl{
		priv:                 priv,
		repo:                 repo,
		reader:               reader,
		cryptographyService:  cryptographyService,
		relayBehaviorService: relayBehaviorService,
		cellRoutingService:   cellRoutingService,
	}
}

func (uc *relayUsecaseRefactoredImpl) ServeConn(c net.Conn) {
	log.Printf("ServeConn start local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	defer func() {
		_ = c.Close()
		log.Printf("ServeConn stop local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	}()

	for {
		cid, cell, err := uc.reader.ReadCell(c)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return
		}
		log.Printf("cell cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
		if err := uc.Handle(c, cid, cell); err != nil {
			log.Println("handle:", err)
		}
	}
}

func (uc *relayUsecaseRefactoredImpl) Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	st, err := uc.repo.Find(cid)
	switch {
	case errors.Is(err, repoif.ErrNotFound) && cell.Cmd == value_object.CmdEnd:
		// End for an unknown circuit is ignored
		return nil
	case errors.Is(err, repoif.ErrNotFound) && cell.Cmd == value_object.CmdExtend:
		// new circuit request
		return uc.extend(up, cid, cell)
	case err != nil:
		return err
	}

	// Use domain services to handle cells based on their type
	switch cell.Cmd {
	case value_object.CmdBegin:
		return uc.handleBeginWithDomainService(st, cid, cell)
	case value_object.CmdBeginAck:
		return uc.forwardCell(st.Up(), cid, cell)
	case value_object.CmdEnd:
		return uc.handleEndWithDomainService(st, cid, cell)
	case value_object.CmdDestroy:
		return uc.handleDestroy(st, cid)
	case value_object.CmdExtend:
		return uc.forwardExtend(st, cid, cell)
	case value_object.CmdConnect:
		return uc.handleConnectWithDomainService(st, cid, cell)
	case value_object.CmdData:
		return uc.handleDataWithDomainService(st, cid, cell)
	default:
		return nil
	}
}

// handleConnectWithDomainService uses domain services to handle CONNECT cells
func (uc *relayUsecaseRefactoredImpl) handleConnectWithDomainService(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	// Delegate complex relay behavior logic to domain service
	instruction, err := uc.relayBehaviorService.HandleConnectCell(st, cell)
	if err != nil {
		return fmt.Errorf("relay behavior service handle connect failed: %w", err)
	}

	return uc.executeInstruction(st, cid, instruction)
}

// handleBeginWithDomainService uses domain services to handle BEGIN cells
func (uc *relayUsecaseRefactoredImpl) handleBeginWithDomainService(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	// Delegate complex relay behavior logic to domain service
	instruction, err := uc.relayBehaviorService.HandleBeginCell(st, cell)
	if err != nil {
		return fmt.Errorf("relay behavior service handle begin failed: %w", err)
	}

	return uc.executeInstruction(st, cid, instruction)
}

// handleDataWithDomainService uses domain services to handle DATA cells
func (uc *relayUsecaseRefactoredImpl) handleDataWithDomainService(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	// Delegate complex relay behavior logic to domain service
	instruction, err := uc.relayBehaviorService.HandleDataCell(st, cell)
	if err != nil {
		return fmt.Errorf("relay behavior service handle data failed: %w", err)
	}

	return uc.executeInstruction(st, cid, instruction)
}

// handleEndWithDomainService uses domain services to handle END cells
func (uc *relayUsecaseRefactoredImpl) handleEndWithDomainService(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	// Delegate to domain service
	instruction, err := uc.relayBehaviorService.HandleEndCell(st, cell)
	if err != nil {
		return fmt.Errorf("relay behavior service handle end failed: %w", err)
	}

	return uc.executeInstruction(st, cid, instruction)
}

// executeInstruction executes the instruction returned by domain services
func (uc *relayUsecaseRefactoredImpl) executeInstruction(st *entity.ConnState, cid value_object.CircuitID, instruction *service.CellHandlingInstruction) error {
	switch instruction.Action {
	case service.ActionForwardDownstream:
		uc.ensureServeDown(st)
		return uc.forwardCell(st.Down(), cid, instruction.ForwardCell)

	case service.ActionForwardUpstream:
		return uc.forwardCell(st.Up(), cid, instruction.ForwardCell)

	case service.ActionEncryptAndForward:
		return uc.forwardCell(st.Up(), cid, instruction.ForwardCell)

	case service.ActionTerminate:
		// Handle locally - could involve stream operations
		if instruction.Response != nil {
			return uc.forwardCell(st.Up(), cid, instruction.Response)
		}
		return nil

	case service.ActionCreateConnection:
		return uc.createConnection(st, cid, &instruction.Connection, instruction.Response)

	default:
		return fmt.Errorf("unknown action: %v", instruction.Action)
	}
}

// createConnection creates a new connection based on connection info
func (uc *relayUsecaseRefactoredImpl) createConnection(st *entity.ConnState, cid value_object.CircuitID, connInfo *service.ConnectionInfo, response *value_object.Cell) error {
	if !connInfo.ShouldDial {
		// Send response without creating connection
		if response != nil {
			return uc.forwardCell(st.Up(), cid, response)
		}
		return nil
	}

	// Create the connection
	down, err := net.Dial("tcp", connInfo.Target)
	if err != nil {
		log.Printf("dial target cid=%s addr=%s err=%v", cid.String(), connInfo.Target, err)
		return err
	}

	// Update connection state
	if st.Down() != nil {
		st.Down().Close()
	}

	beginCounter, dataCounter := st.GetCounters()
	newSt := entity.NewConnStateWithCounters(st.Key(), st.Nonce(), st.Up(), down, beginCounter, dataCounter)
	newSt.SetHidden(connInfo.IsHidden)

	if err := uc.repo.Add(cid, newSt); err != nil {
		down.Close()
		return err
	}

	// Start upstream forwarding for the connection
	if connInfo.StreamID != 0 {
		if err := newSt.Streams().Add(connInfo.StreamID, down); err != nil && !errors.Is(err, entity.ErrDuplicate) {
			down.Close()
			return err
		}
		go uc.forwardUpstream(newSt, cid, connInfo.StreamID, down)
	} else if connInfo.IsHidden {
		go uc.forwardUpstream(newSt, cid, 0, down)
	}

	// Send response
	if response != nil {
		return uc.forwardCell(newSt.Up(), cid, response)
	}

	return nil
}

// ensureServeDown ensures that the downstream connection is being served
func (uc *relayUsecaseRefactoredImpl) ensureServeDown(st *entity.ConnState) {
	if st == nil || st.Down() == nil || st.IsServed() {
		return
	}
	st.MarkServed()
	go uc.ServeConn(st.Down())
}

// forwardCell forwards a cell to the specified connection
func (uc *relayUsecaseRefactoredImpl) forwardCell(w net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	buf, err := value_object.Encode(*cell)
	if err != nil {
		log.Printf("forward encode cid=%s err=%v", cid.String(), err)
		return err
	}
	out := append(cid.Bytes(), buf...)
	_, err = w.Write(out)
	if err != nil {
		log.Printf("forward write cid=%s err=%v", cid.String(), err)
		return err
	}
	log.Printf("response forward cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
	return nil
}

// forwardUpstream forwards upstream data with proper encryption using domain service
func (uc *relayUsecaseRefactoredImpl) forwardUpstream(st *entity.ConnState, cid value_object.CircuitID, sid value_object.StreamID, down net.Conn) {
	defer down.Close()
	buf := make([]byte, value_object.MaxDataLen)
	for {
		n, err := down.Read(buf)
		if n > 0 {
			// Use cryptography domain service for upstream encryption
			encrypted, encErr := uc.cryptographyService.EncryptAtRelay(st, entity.MessageTypeUpstreamData, buf[:n])
			if encErr == nil {
				payload, payloadErr := value_object.EncodeDataPayload(&value_object.DataPayload{
					StreamID: sid.UInt16(),
					Data:     encrypted,
				})
				if payloadErr == nil {
					cell := &value_object.Cell{
						Cmd:     value_object.CmdData,
						Version: value_object.Version,
						Payload: payload,
					}
					_ = uc.forwardCell(st.Up(), cid, cell)
				}
			}
		}
		if err != nil {
			if sid != 0 {
				_ = st.Streams().Remove(sid)
			}
			endPayload := []byte{}
			if sid != 0 {
				endPayload, _ = value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: sid.UInt16()})
			}
			_ = uc.forwardCell(st.Up(), cid, &value_object.Cell{
				Cmd:     value_object.CmdEnd,
				Version: value_object.Version,
				Payload: endPayload,
			})
			return
		}
	}
}

// handleDestroy handles circuit destruction
func (uc *relayUsecaseRefactoredImpl) handleDestroy(st *entity.ConnState, cid value_object.CircuitID) error {
	if st.Down() != nil {
		c := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
		_ = uc.forwardCell(st.Down(), cid, c)
	}
	_ = uc.repo.Delete(cid)
	return nil
}

// Legacy methods for circuit establishment (simplified for this example)
func (uc *relayUsecaseRefactoredImpl) extend(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	// This method remains largely unchanged as it's about circuit establishment
	// which could be extracted to a separate CircuitEstablishmentService in the future
	return fmt.Errorf("extend not implemented in refactored version - would use CircuitTopologyService")
}

func (uc *relayUsecaseRefactoredImpl) forwardExtend(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	if st.Down() == nil {
		return errors.New("no downstream connection")
	}
	if err := uc.forwardCell(st.Down(), cid, cell); err != nil {
		log.Printf("forward extend cid=%s err=%v", cid.String(), err)
		return err
	}
	var hdr [20]byte
	if _, err := io.ReadFull(st.Down(), hdr[:]); err != nil {
		return err
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	if l == 0 {
		return errors.New("malformed created payload")
	}
	payload := make([]byte, l)
	if _, err := io.ReadFull(st.Down(), payload); err != nil {
		return err
	}
	return uc.sendCreated(st.Up(), cid, payload)
}

func (uc *relayUsecaseRefactoredImpl) sendCreated(w net.Conn, cid value_object.CircuitID, payload []byte) error {
	var hdr [20]byte
	copy(hdr[:16], cid.Bytes())
	hdr[16] = value_object.CmdCreated
	hdr[17] = value_object.Version
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	if err != nil {
		return err
	}
	log.Printf("response created cid=%s", cid.String())
	return nil
}
