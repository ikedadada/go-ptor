package usecase

import (
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// RelayUseCase processes cells for a single relay connection.
type RelayUseCase interface {
	Handle(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error
	ServeConn(c net.Conn)
}

type relayUsecaseImpl struct {
	priv   *rsa.PrivateKey
	repo   repository.ConnStateRepository
	crypto service.CryptoService
	crs    service.CellReaderService
}

func (uc *relayUsecaseImpl) ensureServeDown(st *entity.ConnState) {
	if st == nil || st.Down() == nil || st.IsServed() {
		return
	}
	st.MarkServed()
	go uc.ServeConn(st.Down())
}

// NewRelayUseCase returns a use case to process relay connections.
func NewRelayUseCase(priv *rsa.PrivateKey, repo repository.ConnStateRepository, c service.CryptoService, crs service.CellReaderService) RelayUseCase {
	return &relayUsecaseImpl{priv: priv, repo: repo, crypto: c, crs: crs}
}

func (uc *relayUsecaseImpl) ServeConn(c net.Conn) {
	log.Printf("ServeConn start local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	defer func() {
		_ = c.Close()
		log.Printf("ServeConn stop local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	}()

	for {
		cid, cell, err := uc.crs.ReadCell(c)
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

func (uc *relayUsecaseImpl) Handle(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	st, err := uc.repo.Find(cid)
	switch {
	case errors.Is(err, repository.ErrNotFound) && cell.Cmd == vo.CmdEnd:
		// End for an unknown circuit is ignored
		return nil
	case errors.Is(err, repository.ErrNotFound) && cell.Cmd == vo.CmdExtend:
		// new circuit request
		return uc.extend(up, cid, cell)
	case err != nil:
		return err
	}

	switch cell.Cmd {
	case vo.CmdBegin:
		return uc.begin(st, cid, cell)
	case vo.CmdBeginAck:
		return forwardCell(st.Up(), cid, cell)
	case vo.CmdEnd:
		return uc.endStream(st, cid, cell)
	case vo.CmdDestroy:
		if st.Down() != nil {
			c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
			_ = forwardCell(st.Down(), cid, c)
		}
		_ = uc.repo.Delete(cid)
		return nil
	case vo.CmdExtend:
		return uc.forwardExtend(st, cid, cell)
	case vo.CmdConnect:
		return uc.connect(st, cid, cell)
	case vo.CmdData:
		return uc.data(st, cid, cell)
	default:
		return nil
	}
}

func (uc *relayUsecaseImpl) connect(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	// middle relay: peel one layer and forward the remaining ciphertext
	if st.Down() != nil {
		uc.ensureServeDown(st)
		nonce := st.BeginNonce()
		log.Printf("connect decrypt cid=%s nonce=%x", cid.String(), nonce)
		dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
		if err != nil {
			return fmt.Errorf("AESOpen connect cid=%s: %w", cid.String(), err)
		}
		c := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: dec}
		return forwardCell(st.Down(), cid, c)
	}

	// exit relay: decode final payload and connect to the hidden service
	nonce := st.BeginNonce()
	log.Printf("connect exit decrypt cid=%s nonce=%x", cid.String(), nonce)
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
	if err != nil {
		return fmt.Errorf("AESOpen connect cid=%s: %w", cid.String(), err)
	}
	addr := os.Getenv("PTOR_HIDDEN_ADDR")
	if addr == "" {
		addr = os.Getenv("HIDDEN_ADDR")
	}
	if addr == "" {
		addr = "hidden:5000"
	}
	if len(dec) > 0 {
		p, err := vo.DecodeConnectPayload(dec)
		if err != nil {
			return err
		}
		if p.Target != "" {
			addr = p.Target
		}
	}

	down, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("dial hidden cid=%s addr=%s err=%v", cid.String(), addr, err)
		return err
	}
	if st.Down() != nil {
		st.Down().Close()
	}
	beginCounter, dataCounter := st.GetCounters()
	newSt := entity.NewConnStateWithCounters(st.Key(), st.Nonce(), st.Up(), down, beginCounter, dataCounter)
	newSt.SetHidden(true)
	if err := uc.repo.Add(cid, newSt); err != nil {
		down.Close()
		return err
	}
	if err := sendAck(newSt.Up(), cid); err != nil {
		return err
	}
	return nil
}

func (uc *relayUsecaseImpl) begin(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	nonce := st.BeginNonce()
	log.Printf("begin decrypt cid=%s nonce=%x key=%x payloadLen=%d", cid.String(), nonce, st.Key(), len(cell.Payload))
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
	if err != nil {
		log.Printf("AESOpen begin failed cid=%s nonce=%x error=%v", cid.String(), nonce, err)
		return fmt.Errorf("AESOpen begin cid=%s: %w", cid.String(), err)
	}
	log.Printf("begin decrypt success cid=%s decryptedLen=%d", cid.String(), len(dec))

	if st.IsHidden() {
		p, err := vo.DecodeBeginPayload(dec)
		if err != nil {
			return err
		}
		sid, err := vo.StreamIDFrom(p.StreamID)
		if err != nil {
			return err
		}
		go uc.forwardUpstream(st, cid, sid, st.Down())
		return sendAck(st.Up(), cid)
	}

	if st.Down() != nil {
		uc.ensureServeDown(st)
		c := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: dec}
		return forwardCell(st.Down(), cid, c)
	}

	p, err := vo.DecodeBeginPayload(dec)
	if err != nil {
		return err
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	down, err := net.Dial("tcp", p.Target)
	if err != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = forwardCell(st.Up(), cid, c)
		log.Printf("dial begin target cid=%s addr=%s err=%v", cid.String(), p.Target, err)
		return err
	}
	if err := uc.repo.AddStream(cid, sid, down); err != nil {
		down.Close()
		return err
	}
	ack := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}
	if err := forwardCell(st.Up(), cid, ack); err != nil {
		return err
	}
	go uc.forwardUpstream(st, cid, sid, down)
	return nil
}

func (uc *relayUsecaseImpl) data(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	p, err := vo.DecodeDataPayload(cell.Payload)
	if err != nil {
		return err
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}

	// Try to decrypt the data for downstream flow
	nonce := st.DataNonce()
	log.Printf("data decrypt cid=%s nonce=%x key=%x dataLen=%d", cid.String(), nonce, st.Key(), len(p.Data))
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, p.Data)
	if err != nil {
		// If decryption fails and this is a middle relay, it might be upstream data
		// Add our encryption layer and forward upstream
		if st.Down() != nil {
			log.Printf("data decrypt failed, treating as upstream data cid=%s", cid.String())
			// Add encryption layer for upstream flow
			upstreamNonce := st.UpstreamDataNonce()
			log.Printf("upstream encrypt layer cid=%s nonce=%x", cid.String(), upstreamNonce)
			enc, err2 := uc.crypto.AESSeal(st.Key(), upstreamNonce, p.Data)
			if err2 != nil {
				log.Printf("upstream encryption failed cid=%s error=%v", cid.String(), err2)
				return err2
			}
			// Forward with additional encryption layer
			upstreamPayload, err3 := vo.EncodeDataPayload(&vo.DataPayload{StreamID: p.StreamID, Data: enc})
			if err3 != nil {
				return err3
			}
			upstreamCell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: upstreamPayload}
			return forwardCell(st.Up(), cid, upstreamCell)
		}
		log.Printf("AESOpen data failed cid=%s nonce=%x error=%v", cid.String(), nonce, err)
		return fmt.Errorf("AESOpen data cid=%s: %w", cid.String(), err)
	}
	log.Printf("data decrypt success cid=%s decryptedLen=%d", cid.String(), len(dec))

	if st.IsHidden() {
		_, err := st.Down().Write(dec)
		return err
	}

	// middle relay: forward downstream with one layer removed
	if st.Down() != nil {
		uc.ensureServeDown(st)
		payload, err := vo.EncodeDataPayload(&vo.DataPayload{StreamID: p.StreamID, Data: dec})
		if err != nil {
			return err
		}
		c := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}
		return forwardCell(st.Down(), cid, c)
	}

	// exit relay: write plaintext to the local stream
	conn, err := uc.repo.GetStream(cid, sid)
	if err != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = forwardCell(st.Up(), cid, c)
		return nil
	}
	if _, err := conn.Write(dec); err != nil {
		_ = uc.repo.RemoveStream(cid, sid)
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = forwardCell(st.Up(), cid, c)
		return err
	}
	return nil
}

func (uc *relayUsecaseImpl) endStream(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	var p *vo.DataPayload
	var err error
	if len(cell.Payload) > 0 {
		p, err = vo.DecodeDataPayload(cell.Payload)
		if err != nil {
			return err
		}
	} else {
		p = &vo.DataPayload{}
	}
	if p.StreamID == 0 {
		uc.repo.DestroyAllStreams(cid)
		if st.Down() != nil {
			uc.ensureServeDown(st)
			_ = forwardCell(st.Down(), cid, cell)
		}
		_ = uc.repo.Delete(cid)
		return nil
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	_ = uc.repo.RemoveStream(cid, sid)
	if st.Down() != nil {
		uc.ensureServeDown(st)
		return forwardCell(st.Down(), cid, cell)
	}
	return nil
}

func (uc *relayUsecaseImpl) extend(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	p, err := vo.DecodeExtendPayload(cell.Payload)
	if err != nil {
		log.Printf("decode extend payload cid=%s err=%v", cid.String(), err)
		return err
	}
	relayPriv, relayPub, err := uc.crypto.X25519Generate()
	if err != nil {
		return err
	}
	secret, err := uc.crypto.X25519Shared(relayPriv, p.ClientPub[:])
	if err != nil {
		return err
	}
	key, nonce, err := uc.crypto.DeriveKeyNonce(secret)
	if err != nil {
		return err
	}
	var down net.Conn
	if p.NextHop != "" {
		down, err = net.Dial("tcp", p.NextHop)
		if err != nil {
			log.Printf("dial next hop cid=%s hop=%s err=%v", cid.String(), p.NextHop, err)
			return err
		}
	}
	st := entity.NewConnState(key, nonce, up, down)
	if err := uc.repo.Add(cid, st); err != nil {
		return err
	}
	if down != nil {
		// ServeConn will be started when the next downstream-forwarding command arrives
	}
	createdPayload, err := vo.EncodeCreatedPayload(&vo.CreatedPayload{RelayPub: to32(relayPub)})
	if err != nil {
		return err
	}
	return sendCreated(up, cid, createdPayload)
}

func (uc *relayUsecaseImpl) forwardExtend(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	if st.Down() == nil {
		return errors.New("no downstream connection")
	}
	if err := forwardCell(st.Down(), cid, cell); err != nil {
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
	return sendCreated(st.Up(), cid, payload)
}

func to32(b []byte) [32]byte {
	var a [32]byte
	copy(a[:], b)
	return a
}

func sendCreated(w net.Conn, cid vo.CircuitID, payload []byte) error {
	var hdr [20]byte
	copy(hdr[:16], cid.Bytes())
	hdr[16] = byte(vo.CmdCreated)
	hdr[17] = byte(vo.ProtocolV1)
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

func sendAck(w net.Conn, cid vo.CircuitID) error {
	c := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}
	if err := forwardCell(w, cid, c); err != nil {
		return err
	}
	log.Printf("response ack cid=%s", cid.String())
	return nil
}

func forwardCell(w net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	buf, err := entity.Encode(*cell)
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

func (uc *relayUsecaseImpl) forwardUpstream(st *entity.ConnState, cid vo.CircuitID, sid vo.StreamID, down net.Conn) {
	defer down.Close()
	buf := make([]byte, entity.MaxPayloadSize)
	for {
		n, err := down.Read(buf)
		if n > 0 {
			// Use upstream-specific nonce for upstream data encryption
			nonce := st.UpstreamDataNonce()
			log.Printf("upstream encrypt cid=%s nonce=%x", cid.String(), nonce)
			enc, err2 := uc.crypto.AESSeal(st.Key(), nonce, buf[:n])
			if err2 == nil {
				payload, err3 := vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16(), Data: enc})
				if err3 == nil {
					c := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}
					_ = forwardCell(st.Up(), cid, c)
				}
			}
		}
		if err != nil {
			if sid != 0 {
				_ = uc.repo.RemoveStream(cid, sid)
			}
			endPayload := []byte{}
			if sid != 0 {
				endPayload, _ = vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16()})
			}
			_ = forwardCell(st.Up(), cid, &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: endPayload})
			return
		}
	}
}
