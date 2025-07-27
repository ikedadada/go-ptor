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

func TestHandleDataUseCase_DataForwardMiddle(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)

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

	// Create data with one layer of encryption
	plain := []byte("hello")
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	payload, _ := peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: 1, Data: enc})
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Data(st, cid, cell, ensureServeDown) }()

	// Should forward decrypted data downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdData {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	dp, err := peSvc.DecodeDataPayload(fwd.Payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !bytes.Equal(dp.Data, plain) {
		t.Fatalf("payload mismatch")
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("data error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}

func TestHandleDataUseCase_DataExit(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	csRepo.Add(cid, st)

	// Add stream connection
	sid, _ := vo.StreamIDFrom(1)
	local1, local2 := net.Pipe()
	csRepo.AddStream(cid, sid, local1)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create encrypted data
	plain := []byte("hello")
	enc, _ := cSvc.AESSeal(key, nonce, plain)
	payload, _ := peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: sid.UInt16(), Data: enc})
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Data(st, cid, cell, ensureServeDown) }()

	// Should write decrypted data to local stream
	out := make([]byte, len(plain))
	if _, err := io.ReadFull(local2, out); err != nil {
		t.Fatalf("read local: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("payload mismatch: got %q, want %q", out, plain)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("data error: %v", err)
	}

	st.Up().Close()
	local1.Close()
	local2.Close()
}

func TestHandleDataUseCase_DataHidden(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	st.SetHidden(true)
	csRepo.Add(cid, st)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create encrypted data
	data := []byte("hello")
	enc, _ := cSvc.AESSeal(key, nonce, data)
	payload, _ := peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: 1, Data: enc})
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Data(st, cid, cell, ensureServeDown) }()

	// Should write decrypted data to downstream connection
	out := make([]byte, len(data))
	if _, err := io.ReadFull(down2, out); err != nil {
		t.Fatalf("read down: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("payload mismatch")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("data error: %v", err)
	}

	up1.Close()
	down1.Close()
	down2.Close()
}

func TestHandleDataUseCase_DataUpstream(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	csRepo.Add(cid, st)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// Create data that fails decryption (simulating upstream data)
	invalidData := []byte("invalid encrypted data")
	payload, _ := peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: 1, Data: invalidData})
	cell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Data(st, cid, cell, ensureServeDown) }()

	// Should add encryption layer and forward upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read upstream: %v", err)
	}

	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode upstream: %v", err)
	}
	if fwd.Cmd != vo.CmdData {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("data error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}
