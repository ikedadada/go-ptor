package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
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

func TestHandleExtendUseCase_Extend(t *testing.T) {
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleExtendUseCase(priv, csRepo, cSvc, csSvc, peSvc)

	// prepare extend cell
	_, pub, _ := cSvc.X25519Generate()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { ln.Accept() }()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := peSvc.EncodeExtendPayload(&service.ExtendPayloadDTO{NextHop: ln.Addr().String(), ClientPub: pubArr})
	cid := vo.NewCircuitID()
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: payload}

	up1, up2 := net.Pipe()
	errCh := make(chan error, 1)
	go func() { errCh <- uc.Extend(up1, cid, cell) }()

	// Read created response
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(up2, hdr); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if vo.CellCommand(hdr[16]) != vo.CmdCreated {
		t.Fatalf("created cmd %d", hdr[16])
	}
	if hdr[17] != byte(vo.ProtocolV1) {
		t.Fatalf("created version %d", hdr[17])
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	body := make([]byte, l)
	if _, err := io.ReadFull(up2, body); err != nil {
		t.Fatalf("read body: %v", err)
	}

	// ensure entry created with retry
	var st *entity.ConnState
	var err error
	timeout := time.After(100 * time.Millisecond)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for entry creation")
		case <-ticker.C:
			st, err = csRepo.Find(cid)
			if err == nil {
				goto found // Entry created successfully
			}
		}
	}
found:
	if st.Down() != nil {
		st.Down().Close()
	}
	st.Up().Close()

	if err := <-errCh; err != nil {
		t.Fatalf("extend error: %v", err)
	}
}

func TestHandleExtendUseCase_ForwardExtend(t *testing.T) {
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	csRepo := repository.NewConnStateRepository(time.Second)
	cSvc := service.NewCryptoService()
	csSvc := service.NewCellSenderService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewHandleExtendUseCase(priv, csRepo, cSvc, csSvc, peSvc)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, up2 := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	csRepo.Add(cid, st)

	_, pub, _ := cSvc.X25519Generate()
	var pubArr [32]byte
	copy(pubArr[:], pub)
	payload, _ := peSvc.EncodeExtendPayload(&service.ExtendPayloadDTO{ClientPub: pubArr})
	cell := &entity.Cell{Cmd: vo.CmdExtend, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.ForwardExtend(st, cid, cell) }()

	// Should forward the extend cell downstream
	fwd := make([]byte, 528)
	if _, err := io.ReadFull(down2, fwd); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	if vo.CellCommand(fwd[16]) != vo.CmdExtend {
		t.Fatalf("forwarded cmd %d", fwd[16])
	}

	// Send back created response
	created, _ := peSvc.EncodeCreatedPayload(&service.CreatedPayloadDTO{RelayPub: pubArr})
	var hdr [20]byte
	copy(hdr[:16], cid.Bytes())
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(created)))
	down2.Write(hdr[:])
	down2.Write(created)

	// Should forward created response upstream
	var respHdr [20]byte
	if _, err := io.ReadFull(up2, respHdr[:]); err != nil {
		t.Fatalf("read created hdr: %v", err)
	}
	if vo.CellCommand(respHdr[16]) != vo.CmdCreated {
		t.Fatalf("created cmd %d", respHdr[16])
	}
	if respHdr[17] != byte(vo.ProtocolV1) {
		t.Fatalf("created version %d", respHdr[17])
	}
	l := binary.BigEndian.Uint16(respHdr[18:20])
	resp := make([]byte, l)
	if _, err := io.ReadFull(up2, resp); err != nil {
		t.Fatalf("read created body: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("forward extend error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}
