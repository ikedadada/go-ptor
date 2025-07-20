package usecase_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

func TestRelayUseCase_ExtendAndForward(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	// prepare extend cell
	_, pub, _ := cSvc.X25519Generate()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { ln.Accept() }()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{NextHop: ln.Addr().String(), ClientPub: pubArr})
	cid := value_object.NewCircuitID()
	cell := &entity.Cell{Cmd: value_object.CmdExtend, Version: value_object.ProtocolV1, Payload: payload}

	up1, up2 := net.Pipe()
	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(up1, cid, cell) }()

	hdr := make([]byte, 20)
	if _, err := io.ReadFull(up2, hdr); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if value_object.CellCommand(hdr[16]) != value_object.CmdCreated {
		t.Fatalf("created cmd %d", hdr[16])
	}
	if hdr[17] != byte(value_object.ProtocolV1) {
		t.Fatalf("created version %d", hdr[17])
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	body := make([]byte, l)
	if _, err := io.ReadFull(up2, body); err != nil {
		t.Fatalf("read body: %v", err)
	}

	// ensure entry created
	time.Sleep(10 * time.Millisecond)
	st, err := repo.Find(cid)
	if err != nil {
		t.Fatalf("entry not created: %v", err)
	}
	st.Down().Close()
	st.Up().Close()
}

func TestRelayUseCase_ForwardExtendExisting(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	_, pub, _ := cSvc.X25519Generate()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{ClientPub: pubArr})
	cell := &entity.Cell{Cmd: value_object.CmdExtend, Version: value_object.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(up1, cid, cell) }()

	fwd := make([]byte, 528)
	if _, err := io.ReadFull(down2, fwd); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	if value_object.CellCommand(fwd[16]) != value_object.CmdExtend {
		t.Fatalf("forwarded cmd %d", fwd[16])
	}

	created, _ := value_object.EncodeCreatedPayload(&value_object.CreatedPayload{RelayPub: pubArr})
	var hdr [20]byte
	copy(hdr[:16], cid.Bytes())
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(created)))
	down2.Write(hdr[:])
	down2.Write(created)

	var respHdr [20]byte
	if _, err := io.ReadFull(up2, respHdr[:]); err != nil {
		t.Fatalf("read created hdr: %v", err)
	}
	if value_object.CellCommand(respHdr[16]) != value_object.CmdCreated {
		t.Fatalf("created cmd %d", respHdr[16])
	}
	if respHdr[17] != byte(value_object.ProtocolV1) {
		t.Fatalf("created version %d", respHdr[17])
	}
	l := binary.BigEndian.Uint16(respHdr[18:20])
	resp := make([]byte, l)
	if _, err := io.ReadFull(up2, resp); err != nil {
		t.Fatalf("read created body: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}

func TestRelayUseCase_EndUnknown(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	cid := value_object.NewCircuitID()
	cell := &entity.Cell{Cmd: value_object.CmdEnd, Version: value_object.ProtocolV1, Payload: nil}
	if err := uc.Handle(nil, cid, cell); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRelayUseCase_EndStreamNoDown(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)
	sid, _ := value_object.StreamIDFrom(1)
	local1, local2 := net.Pipe()
	st.Streams().Add(sid, local1)

	payload, _ := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: sid.UInt16()})
	cell := &entity.Cell{Cmd: value_object.CmdEnd, Version: value_object.ProtocolV1, Payload: payload}
	if err := uc.Handle(up1, cid, cell); err != nil {
		t.Fatalf("handle: %v", err)
	}

	buf := make([]byte, 1)
	up2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	n, err := up2.Read(buf)
	if n != 0 {
		t.Errorf("unexpected bytes forwarded")
	}
	if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
		t.Errorf("expected timeout, got %v", err)
	}

	local2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	if _, err := local2.Read(buf); err == nil {
		t.Errorf("stream not closed")
	}

	st.Up().Close()
}

func TestRelayUseCase_ForwardEndDestroy(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()

	t.Run("end", func(t *testing.T) {
		up1, _ := net.Pipe()
		down1, down2 := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, down1)
		repo.Add(cid, st)
		cell := &entity.Cell{Cmd: value_object.CmdEnd, Version: value_object.ProtocolV1}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid, cell) }()
		buf := make([]byte, 528)
		if _, err := io.ReadFull(down2, buf); err != nil {
			t.Fatalf("read: %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		if value_object.CellCommand(buf[16]) != value_object.CmdEnd {
			t.Errorf("forwarded cmd %d", buf[16])
		}
		if _, err := repo.Find(cid); err == nil {
			t.Errorf("entry not removed")
		}
	})

	t.Run("destroy", func(t *testing.T) {
		cid2 := value_object.NewCircuitID()
		up1, _ := net.Pipe()
		down1, down2 := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, down1)
		repo.Add(cid2, st)
		cell := &entity.Cell{Cmd: value_object.CmdDestroy, Version: value_object.ProtocolV1}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid2, cell) }()
		buf := make([]byte, 528)
		if _, err := io.ReadFull(down2, buf); err != nil {
			t.Fatalf("read: %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		if value_object.CellCommand(buf[16]) != value_object.CmdDestroy {
			t.Errorf("forwarded cmd %d", buf[16])
		}
		if _, err := repo.Find(cid2); err == nil {
			t.Errorf("entry not removed")
		}
	})
}

func TestRelayUseCase_Connect(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		repo := repoimpl.NewCircuitTableRepository(time.Second)
		cSvc := service.NewCryptoService()
		crSvc := service.NewCellReaderService()
		uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

		key, _ := value_object.NewAESKey()
		nonce, _ := value_object.NewNonce()
		cid := value_object.NewCircuitID()
		up1, up2 := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, nil)
		repo.Add(cid, st)

		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		defer ln.Close()
		go func() {
			c, _ := ln.Accept()
			if c != nil {
				c.Close()
			}
		}()
		payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: ln.Addr().String()})
		enc, _ := cSvc.AESSeal(key, nonce, payload)
		cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: enc}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid, cell) }()
		ack := make([]byte, 16+entity.MaxCellSize)
		if _, err := io.ReadFull(up2, ack); err != nil {
			t.Fatalf("read ack: %v", err)
		}
		cAck, err := entity.Decode(ack[16:])
		if err != nil {
			t.Fatalf("decode ack: %v", err)
		}
		if cAck.Cmd != value_object.CmdBeginAck {
			t.Fatalf("ack cmd %d", cAck.Cmd)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		st2, _ := repo.Find(cid)
		if !st2.IsHidden() {
			t.Errorf("hidden flag not set")
		}
		st2.Up().Close()
		if st2.Down() != nil {
			st2.Down().Close()
		}
	})

	t.Run("env addr", func(t *testing.T) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		repo := repoimpl.NewCircuitTableRepository(time.Second)
		cSvc := service.NewCryptoService()
		crSvc := service.NewCellReaderService()
		uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

		key, _ := value_object.NewAESKey()
		nonce, _ := value_object.NewNonce()
		cid := value_object.NewCircuitID()
		up1, up2 := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, nil)
		repo.Add(cid, st)

		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		defer ln.Close()
		go func() {
			c, _ := ln.Accept()
			if c != nil {
				c.Close()
			}
		}()
		os.Setenv("PTOR_HIDDEN_ADDR", ln.Addr().String())
		defer os.Unsetenv("PTOR_HIDDEN_ADDR")
		enc, _ := cSvc.AESSeal(key, nonce, []byte{})
		cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: enc}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid, cell) }()
		ack := make([]byte, 16+entity.MaxCellSize)
		if _, err := io.ReadFull(up2, ack); err != nil {
			t.Fatalf("read ack: %v", err)
		}
		cAck, err := entity.Decode(ack[16:])
		if err != nil {
			t.Fatalf("decode ack: %v", err)
		}
		if cAck.Cmd != value_object.CmdBeginAck {
			t.Fatalf("ack cmd %d", cAck.Cmd)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		st2, _ := repo.Find(cid)
		if !st2.IsHidden() {
			t.Errorf("hidden flag not set")
		}
		st2.Up().Close()
		if st2.Down() != nil {
			st2.Down().Close()
		}
	})

	t.Run("fail dial", func(t *testing.T) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		repo := repoimpl.NewCircuitTableRepository(time.Second)
		cSvc := service.NewCryptoService()
		crSvc := service.NewCellReaderService()
		uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

		key, _ := value_object.NewAESKey()
		nonce, _ := value_object.NewNonce()
		cid := value_object.NewCircuitID()
		up1, _ := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, nil)
		repo.Add(cid, st)

		payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: "127.0.0.1:1"})
		enc, _ := cSvc.AESSeal(key, nonce, payload)
		cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: enc}
		if err := uc.Handle(up1, cid, cell); err == nil {
			t.Errorf("expected error")
		}
		st.Up().Close()
	})
}

func TestRelayUseCase_ConnectAck(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	connCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); connCh <- c }()

	payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: ln.Addr().String()})
	enc, _ := cSvc.AESSeal(key, nonce, payload)
	cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: enc}
	go uc.Handle(up1, cid, cell)

	ack := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	cAck, err := entity.Decode(ack[16:])
	if err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if cAck.Cmd != value_object.CmdBeginAck {
		t.Fatalf("cmd %d", cAck.Cmd)
	}

	hs := <-connCh
	if hs == nil {
		t.Fatalf("no connection")
	}
	hs.Close()
	up1.Close()
	up2.Close()
}

func TestRelayUseCase_BeginForward(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	plain, _ := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: 1, Target: "example.com:80"})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: value_object.CmdBegin, Version: value_object.ProtocolV1, Payload: enc}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(up1, cid, cell) }()

	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != value_object.CmdBegin {
		t.Fatalf("cmd %d", fwd.Cmd)
	}
	if !bytes.Equal(fwd.Payload, plain) {
		t.Errorf("payload mismatch")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}

func TestRelayUseCase_BeginExit(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	acceptCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); acceptCh <- c }()

	plain, _ := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: 1, Target: ln.Addr().String()})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: value_object.CmdBegin, Version: value_object.ProtocolV1, Payload: enc}

	go uc.Handle(up1, cid, cell)

	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if value_object.CellCommand(buf[16]) != value_object.CmdBeginAck {
		t.Fatalf("ack cmd %d", buf[16])
	}

	c := <-acceptCh
	if c == nil {
		t.Fatalf("no connection")
	}

	c.Close()

	st2, _ := repo.Find(cid)
	sid, _ := value_object.StreamIDFrom(1)
	if _, err := st2.Streams().Get(sid); err != nil {
		t.Fatalf("stream not stored: %v", err)
	}

	st2.Up().Close()
	if st2.Down() != nil {
		st2.Down().Close()
	}
	st2.Streams().DestroyAll()
}

func TestRelayUseCase_DataForwardExit(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	crypto := service.NewCryptoService()

	repoMid := repoimpl.NewCircuitTableRepository(10 * time.Second)
	repoExit := repoimpl.NewCircuitTableRepository(10 * time.Second)

	ucMid := usecase.NewRelayUseCase(priv, repoMid, crypto, service.NewCellReaderService())
	ucExit := usecase.NewRelayUseCase(priv, repoExit, crypto, service.NewCellReaderService())

	keyMid, _ := value_object.NewAESKey()
	nonceMid, _ := value_object.NewNonce()
	keyExit, _ := value_object.NewAESKey()
	nonceExit, _ := value_object.NewNonce()

	cid := value_object.NewCircuitID()

	// middle hop state
	up1, _ := net.Pipe()
	down1, up2 := net.Pipe()
	stMid := entity.NewConnState(keyMid, nonceMid, up1, down1)
	repoMid.Add(cid, stMid)

	// exit hop state
	stExit := entity.NewConnState(keyExit, nonceExit, up2, nil)
	repoExit.Add(cid, stExit)
	sid, _ := value_object.StreamIDFrom(1)
	local1, local2 := net.Pipe()
	stExit.Streams().Add(sid, local1)

	// build data cell as received by middle (layered for exit)
	plain := []byte("hello")
	layerExit, _ := crypto.AESSeal(keyExit, nonceExit, plain)
	layerMid, _ := crypto.AESSeal(keyMid, nonceMid, layerExit)
	payload, _ := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: sid.UInt16(), Data: layerMid})
	cell := &entity.Cell{Cmd: value_object.CmdData, Version: value_object.ProtocolV1, Payload: payload}

	// handle at middle relay
	errCh := make(chan error, 1)
	go func() { errCh <- ucMid.Handle(up1, cid, cell) }()

	// read forwarded cell to exit
	buf := make([]byte, 528)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	dp, err := value_object.DecodeDataPayload(fwd.Payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !bytes.Equal(dp.Data, layerExit) {
		t.Fatalf("forwarded payload mismatch")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle middle: %v", err)
	}

	// exit relay should decrypt and deliver to local connection
	errCh2 := make(chan error, 1)
	go func() { errCh2 <- ucExit.Handle(up2, cid, fwd) }()

	out := make([]byte, len(plain))
	if _, err := io.ReadFull(local2, out); err != nil {
		t.Fatalf("read local: %v", err)
	}
	if string(out) != string(plain) {
		t.Errorf("payload mismatch: %q", out)
	}

	if err := <-errCh2; err != nil {
		t.Fatalf("handle exit: %v", err)
	}

	stMid.Up().Close()
	stMid.Down().Close()
	stExit.Up().Close()
	local1.Close()
	local2.Close()
}

func TestRelayUseCase_ForwardConnectData(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	connCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); connCh <- c }()

	payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: ln.Addr().String()})
	enc, _ := cSvc.AESSeal(key, nonce, payload)
	cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: enc}
	go uc.Handle(up1, cid, cell)

	ack := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	cAck, err := entity.Decode(ack[16:])
	if err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if cAck.Cmd != value_object.CmdBeginAck {
		t.Fatalf("ack cmd %d", cAck.Cmd)
	}

	hs := <-connCh
	if hs == nil {
		t.Fatalf("no connection")
	}

	data := []byte("hello")
	hs.Write(data)

	up2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	n, err := up2.Read(make([]byte, 1))
	if n != 0 {
		t.Fatalf("unexpected data forwarded")
	}
	if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
		t.Fatalf("expected timeout, got %v", err)
	}

	hs.Close()
	up1.Close()
	up2.Close()
}

func TestRelayUseCase_BeginHidden(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	st.SetHidden(true)
	repo.Add(cid, st)

	plain, _ := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: 1, Target: "svc"})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: value_object.CmdBegin, Version: value_object.ProtocolV1, Payload: enc}

	go uc.Handle(up1, cid, cell)

	down2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	if n, err := down2.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Fatalf("unexpected bytes forwarded")
	} else if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
		t.Fatalf("expected timeout, got %v", err)
	}

	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if value_object.CellCommand(buf[16]) != value_object.CmdBeginAck {
		t.Fatalf("ack cmd %d", buf[16])
	}

	// data from hidden service should be forwarded with stream ID
	data := []byte("hi")
	down2.Write(data)
	out := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, out); err != nil {
		t.Fatalf("read data: %v", err)
	}
	c, err := entity.Decode(out[16:])
	if err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if c.Cmd != value_object.CmdData {
		t.Fatalf("cmd %d", c.Cmd)
	}
	dp, err := value_object.DecodeDataPayload(c.Payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	enc2, _ := cSvc.AESSeal(key, nonce, data)
	if dp.StreamID != 1 || !bytes.Equal(dp.Data, enc2) {
		t.Fatalf("payload mismatch")
	}

	up1.Close()
	up2.Close()
	down1.Close()
	down2.Close()
}

func TestRelayUseCase_DataHidden(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	st.SetHidden(true)
	repo.Add(cid, st)

	data := []byte("hello")
	enc, _ := cSvc.AESSeal(key, nonce, data)
	payload, _ := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: 1, Data: enc})
	cell := &entity.Cell{Cmd: value_object.CmdData, Version: value_object.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(up1, cid, cell) }()

	out := make([]byte, len(data))
	if _, err := io.ReadFull(down2, out); err != nil {
		t.Fatalf("read down: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("payload mismatch")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("handle: %v", err)
	}

	up1.Close()
	up2.Close()
	down1.Close()
	down2.Close()
}

func TestRelayUseCase_ForwardAck(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	cell := &entity.Cell{Cmd: value_object.CmdBeginAck, Version: value_object.ProtocolV1}
	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(down1, cid, cell) }()

	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != value_object.CmdBeginAck {
		t.Fatalf("cmd %d", fwd.Cmd)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("handle: %v", err)
	}

	up1.Close()
	up2.Close()
	down1.Close()
	down2.Close()
}

func TestRelayUseCase_MultiHopExtend(t *testing.T) {
	priv1, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv2, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo1 := repoimpl.NewCircuitTableRepository(time.Second)
	repo2 := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()

	uc1 := usecase.NewRelayUseCase(priv1, repo1, cSvc, crSvc)
	uc2 := usecase.NewRelayUseCase(priv2, repo2, cSvc, crSvc)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			go uc2.ServeConn(conn)
		}
	}()

	_, pub1, _ := cSvc.X25519Generate()
	var pubArr1 [32]byte
	copy(pubArr1[:], pub1)
	payload1, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{NextHop: ln.Addr().String(), ClientPub: pubArr1})
	cid := value_object.NewCircuitID()
	cell1 := &entity.Cell{Cmd: value_object.CmdExtend, Version: value_object.ProtocolV1, Payload: payload1}

	upEntry, upClient := net.Pipe()
	go uc1.Handle(upEntry, cid, cell1)

	hdr := make([]byte, 20)
	if _, err := io.ReadFull(upClient, hdr); err != nil {
		t.Fatalf("read created1 hdr: %v", err)
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	buf1 := make([]byte, l)
	if _, err := io.ReadFull(upClient, buf1); err != nil {
		t.Fatalf("read created1 body: %v", err)
	}

	_, pub2, _ := cSvc.X25519Generate()
	var pubArr2 [32]byte
	copy(pubArr2[:], pub2)
	payload2, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{ClientPub: pubArr2})
	cell2 := &entity.Cell{Cmd: value_object.CmdExtend, Version: value_object.ProtocolV1, Payload: payload2}
	go uc1.Handle(upEntry, cid, cell2)

	hdr2 := make([]byte, 20)
	if _, err := io.ReadFull(upClient, hdr2); err != nil {
		t.Fatalf("read created2 hdr: %v", err)
	}
	l2 := binary.BigEndian.Uint16(hdr2[18:20])
	buf2 := make([]byte, l2)
	if _, err := io.ReadFull(upClient, buf2); err != nil {
		t.Fatalf("read created2 body: %v", err)
	}
	if value_object.CellCommand(hdr2[16]) != value_object.CmdCreated {
		t.Fatalf("second created cmd %d", hdr2[16])
	}
	upEntry.Close()
	upClient.Close()
}

func TestRelayUseCase_AESOpenErrorContext(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	uc := usecase.NewRelayUseCase(priv, repo, cSvc, crSvc)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, _ := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	cell := &entity.Cell{Cmd: value_object.CmdConnect, Version: value_object.ProtocolV1, Payload: []byte{1, 2, 3}}
	err := uc.Handle(up1, cid, cell)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "connect") || !strings.Contains(err.Error(), cid.String()) {
		t.Fatalf("missing context: %v", err)
	}
	st.Up().Close()
}
