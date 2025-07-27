package usecase_test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleBeginUseCase_BeginForward(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	csRepo.Add(cid, st)

	// Mock ensureServeDown function
	serveDownCalled := false
	ensureServeDown := func(st *entity.ConnState) {
		serveDownCalled = true
	}

	plain, _ := peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: 1, Target: "example.com:80"})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: enc}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Begin(st, cid, cell, ensureServeDown) }()

	// Should forward the decrypted begin cell downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdBegin {
		t.Fatalf("cmd %d", fwd.Cmd)
	}
	if !bytes.Equal(fwd.Payload, plain) {
		t.Errorf("payload mismatch")
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("begin error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}

func TestHandleBeginUseCase_BeginExit(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	csRepo.Add(cid, st)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer ln.Close()
	acceptCh := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			t.Logf("Accept failed: %v", err)
		}
		acceptCh <- c
	}()

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	target := ln.Addr().String()
	t.Logf("Target address: %s", target)
	plain, _ := peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: 1, Target: target})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: enc}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Begin(st, cid, cell, ensureServeDown) }()

	// Should send ack upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if vo.CellCommand(buf[16]) != vo.CmdBeginAck {
		t.Fatalf("ack cmd %d", buf[16])
	}

	// Wait for Begin operation to complete first
	if err := <-errCh; err != nil {
		t.Fatalf("begin error: %v", err)
	}

	// Should store stream in repository
	sid, _ := vo.StreamIDFrom(1)
	if _, err := csRepo.GetStream(cid, sid); err != nil {
		t.Fatalf("stream not stored: %v", err)
	}

	// Should establish connection to target
	c := <-acceptCh
	if c == nil {
		t.Fatalf("no connection")
	}
	c.Close()

	st.Up().Close()
	csRepo.DestroyAllStreams(cid)
}

func TestHandleBeginUseCase_BeginHidden(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	st.SetHidden(true)
	csRepo.Add(cid, st)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	plain, _ := peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: 1, Target: "svc"})
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	cell := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: enc}

	go uc.Begin(st, cid, cell, ensureServeDown)

	// Should not forward downstream when hidden
	down2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	if n, err := down2.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Fatalf("unexpected bytes forwarded")
	} else if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
		t.Fatalf("expected timeout, got %v", err)
	}

	// Should send ack upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if vo.CellCommand(buf[16]) != vo.CmdBeginAck {
		t.Fatalf("ack cmd %d", buf[16])
	}

	up1.Close()
	up2.Close()
	down1.Close()
	down2.Close()
}
