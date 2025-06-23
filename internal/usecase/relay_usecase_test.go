package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
)

func TestRelayUseCase_ExtendAndForward(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)

	// prepare extend cell
	_, pub, _ := crypto.X25519Generate()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { ln.Accept() }()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{NextHop: ln.Addr().String(), ClientPub: pubArr})
	cid := value_object.NewCircuitID()
	cell := &value_object.Cell{Cmd: value_object.CmdExtend, Version: value_object.Version, Payload: payload}

	up1, up2 := net.Pipe()
	go uc.Handle(up1, cid, cell)

	hdr := make([]byte, 20)
	if _, err := io.ReadFull(up2, hdr); err != nil {
		t.Fatalf("read header: %v", err)
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
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	_, pub, _ := crypto.X25519Generate()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{ClientPub: pubArr})
	cell := &value_object.Cell{Cmd: value_object.CmdExtend, Version: value_object.Version, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Handle(up1, cid, cell) }()

	fwd := make([]byte, 528)
	if _, err := io.ReadFull(down2, fwd); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	if fwd[16] != value_object.CmdExtend {
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
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)
	cid := value_object.NewCircuitID()
	cell := &value_object.Cell{Cmd: value_object.CmdEnd, Version: value_object.Version, Payload: nil}
	if err := uc.Handle(nil, cid, cell); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRelayUseCase_ForwardEndDestroy(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()

	t.Run("end", func(t *testing.T) {
		up1, _ := net.Pipe()
		down1, down2 := net.Pipe()
		st := entity.NewConnState(key, nonce, up1, down1)
		repo.Add(cid, st)
		cell := &value_object.Cell{Cmd: value_object.CmdEnd, Version: value_object.Version}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid, cell) }()
		buf := make([]byte, 528)
		if _, err := io.ReadFull(down2, buf); err != nil {
			t.Fatalf("read: %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		if buf[16] != value_object.CmdEnd {
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
		cell := &value_object.Cell{Cmd: value_object.CmdDestroy, Version: value_object.Version}
		errCh := make(chan error, 1)
		go func() { errCh <- uc.Handle(up1, cid2, cell) }()
		buf := make([]byte, 528)
		if _, err := io.ReadFull(down2, buf); err != nil {
			t.Fatalf("read: %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("handle: %v", err)
		}
		if buf[16] != value_object.CmdDestroy {
			t.Errorf("forwarded cmd %d", buf[16])
		}
		if _, err := repo.Find(cid2); err == nil {
			t.Errorf("entry not removed")
		}
	})
}

func TestRelayUseCase_Connect(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)

	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	cid := value_object.NewCircuitID()
	up1, up2 := net.Pipe()
	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	t.Run("ok", func(t *testing.T) {
		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		defer ln.Close()
		go func() {
			c, _ := ln.Accept()
			if c != nil {
				c.Close()
			}
		}()
		payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: ln.Addr().String()})
		cell := &value_object.Cell{Cmd: value_object.CmdConnect, Version: value_object.Version, Payload: payload}
		go uc.Handle(up1, cid, cell)
		buf := make([]byte, 20)
		if _, err := io.ReadFull(up2, buf); err != nil {
			t.Fatalf("read ack: %v", err)
		}
	})

	t.Run("fail dial", func(t *testing.T) {
		payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: "127.0.0.1:1"})
		cell := &value_object.Cell{Cmd: value_object.CmdConnect, Version: value_object.Version, Payload: payload}
		if err := uc.Handle(up1, cid, cell); err == nil {
			t.Errorf("expected error")
		}
	})

	st2, _ := repo.Find(cid)
	st2.Up().Close()
	if st2.Down() != nil {
		st2.Down().Close()
	}
}
