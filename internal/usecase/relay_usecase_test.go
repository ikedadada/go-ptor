package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
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
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	msg := append(key[:], nonce[:]...)
	enc, _ := crypto.RSAEncrypt(&priv.PublicKey, msg)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { ln.Accept() }()
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{NextHop: ln.Addr().String(), EncKey: enc})
	cid := value_object.NewCircuitID()
	cell := &value_object.Cell{Cmd: value_object.CmdExtend, Version: value_object.Version, Payload: payload}

	up1, up2 := net.Pipe()
	go uc.Handle(up1, cid, cell)

	buf := make([]byte, 20)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
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
