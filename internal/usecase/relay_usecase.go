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

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// RelayUseCase processes cells for a single relay connection.
type RelayUseCase interface {
	Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error
	ServeConn(c net.Conn)
}

type relayUsecaseImpl struct {
	priv   *rsa.PrivateKey
	repo   repoif.CircuitTableRepository
	crypto service.CryptoService
	reader service.CellReader
}

func (uc *relayUsecaseImpl) ensureServeDown(st *entity.ConnState) {
	if st == nil || st.Down() == nil || st.IsServed() {
		return
	}
	st.MarkServed()
	go uc.ServeConn(st.Down())
}

// NewRelayUseCase returns a use case to process relay connections.
func NewRelayUseCase(priv *rsa.PrivateKey, repo repoif.CircuitTableRepository, c service.CryptoService, r service.CellReader) RelayUseCase {
	return &relayUsecaseImpl{priv: priv, repo: repo, crypto: c, reader: r}
}

func (uc *relayUsecaseImpl) ServeConn(c net.Conn) {
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

func (uc *relayUsecaseImpl) Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
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

	switch cell.Cmd {
	case value_object.CmdBegin:
		return uc.begin(st, cid, cell)
	case value_object.CmdBeginAck:
		return forwardCell(st.Up(), cid, cell)
	case value_object.CmdEnd:
		return uc.endStream(st, cid, cell)
	case value_object.CmdDestroy:
		if st.Down() != nil {
			c := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
			_ = forwardCell(st.Down(), cid, c)
		}
		_ = uc.repo.Delete(cid)
		return nil
	case value_object.CmdExtend:
		return uc.forwardExtend(st, cid, cell)
	case value_object.CmdConnect:
		return uc.connect(st, cid, cell)
	case value_object.CmdData:
		return uc.data(st, cid, cell)
	default:
		return nil
	}
}

func (uc *relayUsecaseImpl) connect(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	// middle relay: peel one layer and forward the remaining ciphertext
	if st.Down() != nil {
		uc.ensureServeDown(st)
		nonce := st.BeginNonce()
		log.Printf("connect decrypt cid=%s nonce=%x", cid.String(), nonce)
		dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
		if err != nil {
			return fmt.Errorf("AESOpen connect cid=%s: %w", cid.String(), err)
		}
		c := &value_object.Cell{Cmd: value_object.CmdConnect, Version: value_object.Version, Payload: dec}
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
		p, err := value_object.DecodeConnectPayload(dec)
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
	newSt := entity.NewConnState(st.Key(), st.Nonce(), st.Up(), down)
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

func (uc *relayUsecaseImpl) begin(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	nonce := st.BeginNonce()
	log.Printf("begin decrypt cid=%s nonce=%x", cid.String(), nonce)
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
	if err != nil {
		return fmt.Errorf("AESOpen begin cid=%s: %w", cid.String(), err)
	}

	if st.IsHidden() {
		p, err := value_object.DecodeBeginPayload(dec)
		if err != nil {
			return err
		}
		sid, err := value_object.StreamIDFrom(p.StreamID)
		if err != nil {
			return err
		}
		go uc.forwardUpstream(st, cid, sid, st.Down())
		return sendAck(st.Up(), cid)
	}

	if st.Down() != nil {
		uc.ensureServeDown(st)
		c := &value_object.Cell{Cmd: value_object.CmdBegin, Version: value_object.Version, Payload: dec}
		return forwardCell(st.Down(), cid, c)
	}

	p, err := value_object.DecodeBeginPayload(dec)
	if err != nil {
		return err
	}
	sid, err := value_object.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	down, err := net.Dial("tcp", p.Target)
	if err != nil {
		c := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
		_ = forwardCell(st.Up(), cid, c)
		log.Printf("dial begin target cid=%s addr=%s err=%v", cid.String(), p.Target, err)
		return err
	}
	if err := st.Streams().Add(sid, down); err != nil {
		if errors.Is(err, entity.ErrDuplicate) {
			_ = st.Streams().Remove(sid)
			_ = st.Streams().Add(sid, down)
		} else {
			down.Close()
			return err
		}
	}
	ack := &value_object.Cell{Cmd: value_object.CmdBeginAck, Version: value_object.Version}
	if err := forwardCell(st.Up(), cid, ack); err != nil {
		return err
	}
	go uc.forwardUpstream(st, cid, sid, down)
	return nil
}

func (uc *relayUsecaseImpl) data(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	p, err := value_object.DecodeDataPayload(cell.Payload)
	if err != nil {
		return err
	}
	sid, err := value_object.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	nonce := st.DataNonce()
	log.Printf("data decrypt cid=%s nonce=%x", cid.String(), nonce)
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, p.Data)
	if err != nil {
		return fmt.Errorf("AESOpen data cid=%s: %w", cid.String(), err)
	}

	if st.IsHidden() {
		_, err := st.Down().Write(dec)
		return err
	}

	// middle relay: forward downstream with one layer removed
	if st.Down() != nil {
		uc.ensureServeDown(st)
		payload, err := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: p.StreamID, Data: dec})
		if err != nil {
			return err
		}
		c := &value_object.Cell{Cmd: value_object.CmdData, Version: value_object.Version, Payload: payload}
		return forwardCell(st.Down(), cid, c)
	}

	// exit relay: write plaintext to the local stream
	conn, err := st.Streams().Get(sid)
	if err != nil {
		c := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
		_ = forwardCell(st.Up(), cid, c)
		return nil
	}
	if _, err := conn.Write(dec); err != nil {
		_ = st.Streams().Remove(sid)
		c := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
		_ = forwardCell(st.Up(), cid, c)
		return err
	}
	return nil
}

func (uc *relayUsecaseImpl) endStream(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	var p *value_object.DataPayload
	var err error
	if len(cell.Payload) > 0 {
		p, err = value_object.DecodeDataPayload(cell.Payload)
		if err != nil {
			return err
		}
	} else {
		p = &value_object.DataPayload{}
	}
	if p.StreamID == 0 {
		st.Streams().DestroyAll()
		if st.Down() != nil {
			uc.ensureServeDown(st)
			_ = forwardCell(st.Down(), cid, cell)
		}
		_ = uc.repo.Delete(cid)
		return nil
	}
	sid, err := value_object.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	_ = st.Streams().Remove(sid)
	if st.Down() != nil {
		uc.ensureServeDown(st)
		return forwardCell(st.Down(), cid, cell)
	}
	return nil
}

func (uc *relayUsecaseImpl) extend(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	p, err := value_object.DecodeExtendPayload(cell.Payload)
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
	createdPayload, err := value_object.EncodeCreatedPayload(&value_object.CreatedPayload{RelayPub: to32(relayPub)})
	if err != nil {
		return err
	}
	return sendCreated(up, cid, createdPayload)
}

func (uc *relayUsecaseImpl) forwardExtend(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
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

func sendCreated(w net.Conn, cid value_object.CircuitID, payload []byte) error {
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

func sendAck(w net.Conn, cid value_object.CircuitID) error {
	c := &value_object.Cell{Cmd: value_object.CmdBeginAck, Version: value_object.Version}
	if err := forwardCell(w, cid, c); err != nil {
		return err
	}
	log.Printf("response ack cid=%s", cid.String())
	return nil
}

func forwardCell(w net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
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

func (uc *relayUsecaseImpl) forwardUpstream(st *entity.ConnState, cid value_object.CircuitID, sid value_object.StreamID, down net.Conn) {
	defer down.Close()
	buf := make([]byte, value_object.MaxDataLen)
	for {
		n, err := down.Read(buf)
		if n > 0 {
			nonce := st.DataNonce()
			log.Printf("upstream encrypt cid=%s nonce=%x", cid.String(), nonce)
			enc, err2 := uc.crypto.AESSeal(st.Key(), nonce, buf[:n])
			if err2 == nil {
				payload, err3 := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: sid.UInt16(), Data: enc})
				if err3 == nil {
					c := &value_object.Cell{Cmd: value_object.CmdData, Version: value_object.Version, Payload: payload}
					_ = forwardCell(st.Up(), cid, c)
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
			_ = forwardCell(st.Up(), cid, &value_object.Cell{Cmd: value_object.CmdEnd, Version: value_object.Version, Payload: endPayload})
			return
		}
	}
}
