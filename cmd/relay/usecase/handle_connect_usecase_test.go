package usecase_test

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleConnectUseCase_ConnectMiddle(t *testing.T) {
	repo := repository.NewConnStateRepository(time.Second)
	crypto := service.NewCryptoService()
	cellSender := service.NewCellSenderService()
	p := service.NewPayloadEncodingService()
	uc := usecase.NewHandleConnectUseCase(repo, crypto, cellSender, p)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	// Create connect payload
	payload, _ := p.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: "example.com:80"})
	enc, _ := crypto.AESSeal(key, nonce, payload)
	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: enc}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Connect(st, cid, cell, ensureServeDown) }()

	// Should forward decrypted connect cell downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdConnect {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("connect error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}

func TestHandleConnectUseCase_ConnectExit(t *testing.T) {
	repo := repository.NewConnStateRepository(time.Second)
	crypto := service.NewCryptoService()
	cellSender := service.NewCellSenderService()
	p := service.NewPayloadEncodingService()
	uc := usecase.NewHandleConnectUseCase(repo, crypto, cellSender, p)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	// Setup mock hidden service
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	connCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); connCh <- c }()

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create connect payload with target
	payload, _ := p.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: ln.Addr().String()})
	enc, _ := crypto.AESSeal(key, nonce, payload)
	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: enc}

	go uc.Connect(st, cid, cell, ensureServeDown)

	// Should send ack upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	cAck, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if cAck.Cmd != vo.CmdBeginAck {
		t.Fatalf("ack cmd %d", cAck.Cmd)
	}

	// Should establish connection to hidden service
	hs := <-connCh
	if hs == nil {
		t.Fatalf("no connection")
	}
	hs.Close()

	// Should update state to hidden
	st2, _ := repo.Find(cid)
	if !st2.IsHidden() {
		t.Errorf("hidden flag not set")
	}

	st2.Up().Close()
	if st2.Down() != nil {
		st2.Down().Close()
	}
}

func TestHandleConnectUseCase_ConnectExitWithEnvAddr(t *testing.T) {
	repo := repository.NewConnStateRepository(time.Second)
	crypto := service.NewCryptoService()
	cellSender := service.NewCellSenderService()
	p := service.NewPayloadEncodingService()
	uc := usecase.NewHandleConnectUseCase(repo, crypto, cellSender, p)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	// Setup mock hidden service
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	connCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); connCh <- c }()

	// Set environment variable
	os.Setenv("PTOR_HIDDEN_ADDR", ln.Addr().String())
	defer os.Unsetenv("PTOR_HIDDEN_ADDR")

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create connect payload with empty payload (should use env var)
	enc, _ := crypto.AESSeal(key, nonce, []byte{})
	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: enc}

	go uc.Connect(st, cid, cell, ensureServeDown)

	// Should send ack upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	cAck, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if cAck.Cmd != vo.CmdBeginAck {
		t.Fatalf("ack cmd %d", cAck.Cmd)
	}

	// Should establish connection to hidden service
	hs := <-connCh
	if hs == nil {
		t.Fatalf("no connection")
	}
	hs.Close()

	// Should update state to hidden
	st2, _ := repo.Find(cid)
	if !st2.IsHidden() {
		t.Errorf("hidden flag not set")
	}

	st2.Up().Close()
	if st2.Down() != nil {
		st2.Down().Close()
	}
}

func TestHandleConnectUseCase_ConnectExitDialFail(t *testing.T) {
	repo := repository.NewConnStateRepository(time.Second)
	crypto := service.NewCryptoService()
	cellSender := service.NewCellSenderService()
	p := service.NewPayloadEncodingService()
	uc := usecase.NewHandleConnectUseCase(repo, crypto, cellSender, p)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create connect payload with invalid target
	payload, _ := p.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: "127.0.0.1:1"})
	enc, _ := crypto.AESSeal(key, nonce, payload)
	cell := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: enc}

	// Should return error when dial fails
	if err := uc.Connect(st, cid, cell, ensureServeDown); err == nil {
		t.Errorf("expected error when dial fails")
	}

	st.Up().Close()
}
