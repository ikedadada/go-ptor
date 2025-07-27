package handler_test

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/cmd/relay/handler"
	"ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestRelayHandler_HandleCellExtend(t *testing.T) {
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	repo := repository.NewConnStateRepository(time.Second)
	crypto := service.NewCryptoService()
	reader := service.NewCellReaderService()

	// Create cell sender and usecases
	cellSender := service.NewCellSenderService()
	payloadEncoder := service.NewPayloadEncodingService()
	extendUC := usecase.NewHandleExtendUseCase(priv, repo, crypto, cellSender, payloadEncoder)
	beginUC := usecase.NewHandleBeginUseCase(repo, crypto, cellSender, payloadEncoder)
	dataUC := usecase.NewHandleDataUseCase(repo, crypto, cellSender, payloadEncoder)
	endStreamUC := usecase.NewHandleEndStreamUseCase(repo, cellSender, payloadEncoder)
	destroyUC := usecase.NewHandleDestroyUseCase(repo, cellSender)
	connectUC := usecase.NewHandleConnectUseCase(repo, crypto, cellSender, payloadEncoder)

	h := handler.NewRelayHandler(repo, reader, cellSender, extendUC, beginUC, dataUC, endStreamUC, destroyUC, connectUC)

	// Create extend cell
	_, pub, _ := crypto.X25519Generate()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := payloadEncoder.EncodeExtendPayload(&service.ExtendPayloadDTO{ClientPub: pubArr})
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: payload}

	up1, up2 := net.Pipe()
	errCh := make(chan error, 1)
	go func() { errCh <- h.HandleCell(up1, cid, cell) }()

	// Should create circuit and send created response
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(up2, hdr); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if vo.CellCommand(hdr[16]) != vo.CmdCreated {
		t.Fatalf("expected created, got %d", hdr[16])
	}
	// Read payload
	l := int(hdr[18])<<8 | int(hdr[19])
	if l > 0 {
		payload := make([]byte, l)
		if _, err := io.ReadFull(up2, payload); err != nil {
			t.Fatalf("read payload: %v", err)
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle cell error: %v", err)
	}

	// Circuit should be created
	if _, err := repo.Find(cid); err != nil {
		t.Fatalf("circuit not created: %v", err)
	}

	up1.Close()
	up2.Close()
}

func TestRelayHandler_HandleCellBeginAck(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()

	// Create dummy usecases (not used for this test)
	extendUC := usecase.NewHandleExtendUseCase(nil, csRepo, cSvc, csSvc, peSvc)
	beginUC := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)
	dataUC := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)
	endStreamUC := usecase.NewHandleEndStreamUseCase(csRepo, csSvc, peSvc)
	destroyUC := usecase.NewHandleDestroyUseCase(csRepo, csSvc)
	connectUC := usecase.NewHandleConnectUseCase(csRepo, cSvc, csSvc, peSvc)

	h := handler.NewRelayHandler(csRepo, crSvc, csSvc, extendUC, beginUC, dataUC, endStreamUC, destroyUC, connectUC)

	// Create state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	csRepo.Add(cid, st)

	// Create begin ack cell
	cell := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}

	errCh := make(chan error, 1)
	go func() { errCh <- h.HandleCell(up1, cid, cell) }()

	// Should forward cell upstream
	buf := make([]byte, 16+entity.MaxCellSize)
	if _, err := io.ReadFull(up2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdBeginAck {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle cell error: %v", err)
	}

	st.Up().Close()
}

func TestRelayHandler_HandleCellDestroy(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()

	// Create dummy usecases (not used for this test)
	extendUC := usecase.NewHandleExtendUseCase(nil, csRepo, cSvc, csSvc, peSvc)
	beginUC := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)
	dataUC := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)
	endStreamUC := usecase.NewHandleEndStreamUseCase(csRepo, csSvc, peSvc)
	destroyUC := usecase.NewHandleDestroyUseCase(csRepo, csSvc)
	connectUC := usecase.NewHandleConnectUseCase(csRepo, cSvc, csSvc, peSvc)

	h := handler.NewRelayHandler(csRepo, crSvc, csSvc, extendUC, beginUC, dataUC, endStreamUC, destroyUC, connectUC)

	// Create state
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	csRepo.Add(cid, st)

	// Create destroy cell
	cell := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}

	errCh := make(chan error, 1)
	go func() { errCh <- h.HandleCell(up1, cid, cell) }()

	// Should forward destroy cell downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdDestroy {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("handle cell error: %v", err)
	}

	// Circuit should be deleted
	if _, err := csRepo.Find(cid); err == nil {
		t.Errorf("circuit not deleted")
	}

	st.Up().Close()
	st.Down().Close()
}

func TestRelayHandler_HandleCellEndUnknown(t *testing.T) {
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()

	// Create dummy usecases (not used for this test)
	extendUC := usecase.NewHandleExtendUseCase(nil, csRepo, cSvc, csSvc, peSvc)
	beginUC := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)
	dataUC := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)
	endStreamUC := usecase.NewHandleEndStreamUseCase(csRepo, csSvc, peSvc)
	destroyUC := usecase.NewHandleDestroyUseCase(csRepo, csSvc)
	connectUC := usecase.NewHandleConnectUseCase(csRepo, cSvc, csSvc, peSvc)

	h := handler.NewRelayHandler(csRepo, crSvc, csSvc, extendUC, beginUC, dataUC, endStreamUC, destroyUC, connectUC)

	// Create end cell for unknown circuit
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: nil}

	// Should ignore end for unknown circuit
	if err := h.HandleCell(nil, cid, cell); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRelayHandler_ServeConn(t *testing.T) {
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	crSvc := service.NewCellReaderService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()

	// Create dummy usecases (not used for this test)
	extendUC := usecase.NewHandleExtendUseCase(priv, csRepo, cSvc, csSvc, peSvc)
	beginUC := usecase.NewHandleBeginUseCase(csRepo, cSvc, csSvc, peSvc)
	dataUC := usecase.NewHandleDataUseCase(csRepo, cSvc, csSvc, peSvc)
	endStreamUC := usecase.NewHandleEndStreamUseCase(csRepo, csSvc, peSvc)
	destroyUC := usecase.NewHandleDestroyUseCase(csRepo, csSvc)
	connectUC := usecase.NewHandleConnectUseCase(csRepo, cSvc, csSvc, peSvc)

	h := handler.NewRelayHandler(csRepo, crSvc, csSvc, extendUC, beginUC, dataUC, endStreamUC, destroyUC, connectUC)

	// Create pipe connection
	conn1, conn2 := net.Pipe()

	// Channel to signal goroutine completion
	done := make(chan struct{})

	// Start serving connection
	go func() {
		defer close(done)
		h.ServeConn(conn1)
	}()

	// Send extend cell
	_, pub, _ := cSvc.X25519Generate()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := peSvc.EncodeExtendPayload(&service.ExtendPayloadDTO{ClientPub: pubArr})
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: payload}
	cellData, _ := entity.Encode(*cell)
	fullCell := append(cid.Bytes(), cellData...)

	_, err := conn2.Write(fullCell)
	if err != nil {
		t.Fatalf("write cell: %v", err)
	}

	// Should receive created response
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(conn2, hdr); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if vo.CellCommand(hdr[16]) != vo.CmdCreated {
		t.Fatalf("expected created, got %d", hdr[16])
	}
	// Read payload
	l := int(hdr[18])<<8 | int(hdr[19])
	if l > 0 {
		payload := make([]byte, l)
		if _, err := io.ReadFull(conn2, payload); err != nil {
			t.Fatalf("read payload: %v", err)
		}
	}

	conn1.Close()
	conn2.Close()

	// Wait for goroutine to complete with timeout
	select {
	case <-done:
		// ServeConn completed successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for ServeConn to complete")
	}
}
