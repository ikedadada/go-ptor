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

// TestTwoHop_BeginDataRoundtrip sets up two relays connected via net.Pipe
// and verifies that a BEGIN and DATA sequence reaches the final hop without
// AESOpen errors.
func TestTwoHop_BeginDataRoundtrip(t *testing.T) {
	priv1, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv2, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo1 := repoimpl.NewCircuitTableRepository(10 * time.Second)
	repo2 := repoimpl.NewCircuitTableRepository(10 * time.Second)
	crypto := infraSvc.NewCryptoService()
	uc1 := usecase.NewRelayUseCase(priv1, repo1, crypto, infraSvc.NewHandlerCellReader())
	uc2 := usecase.NewRelayUseCase(priv2, repo2, crypto, infraSvc.NewHandlerCellReader())

	key1, _ := value_object.NewAESKey()
	nonce1, _ := value_object.NewNonce()
	key2, _ := value_object.NewAESKey()
	nonce2, _ := value_object.NewNonce()

	cid := value_object.NewCircuitID()

	client, entry := net.Pipe()
	down1, up2 := net.Pipe()

	stEntry := entity.NewConnState(key1, nonce1, entry, down1)
	repo1.Add(cid, stEntry)
	stExit := entity.NewConnState(key2, nonce2, up2, nil)
	repo2.Add(cid, stExit)

	go uc2.ServeConn(up2)
	go uc1.ServeConn(entry)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	acceptCh := make(chan net.Conn, 1)
	go func() { c, _ := ln.Accept(); acceptCh <- c }()

	sid, _ := value_object.StreamIDFrom(1)
	plainBegin, _ := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: sid.UInt16(), Target: ln.Addr().String()})
	layer2, _ := crypto.AESSeal(key2, nonce2, plainBegin)
	layer1, _ := crypto.AESSeal(key1, nonce1, layer2)
	cellBegin := &value_object.Cell{Cmd: value_object.CmdBegin, Version: value_object.Version, Payload: layer1}
	buf, _ := value_object.Encode(*cellBegin)
	client.Write(append(cid.Bytes(), buf...))

	hs := <-acceptCh
	if hs == nil {
		t.Fatalf("hidden service not connected")
	}

	data := []byte("hello")
	enc2, _ := crypto.AESSeal(key2, nonce2, data)
	enc1, _ := crypto.AESSeal(key1, nonce1, enc2)
	payload, _ := value_object.EncodeDataPayload(&value_object.DataPayload{StreamID: sid.UInt16(), Data: enc1})
	cellData := &value_object.Cell{Cmd: value_object.CmdData, Version: value_object.Version, Payload: payload}
	buf, _ = value_object.Encode(*cellData)
	client.Write(append(cid.Bytes(), buf...))

	out := make([]byte, len(data))
	if _, err := io.ReadFull(hs, out); err != nil {
		t.Fatalf("read hidden: %v", err)
	}
	if string(out) != string(data) {
		t.Errorf("payload mismatch: %q", out)
	}

	client.Close()
	entry.Close()
	down1.Close()
	up2.Close()
	hs.Close()
}
