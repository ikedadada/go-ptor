package usecase_test

import (
	"io"
	"net"
	"testing"
	"time"

	repoimpl "ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleEndStreamUseCase_EndStreamSpecific(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	cellSender := service.NewCellSenderService()
	uc := usecase.NewHandleEndStreamUseCase(repo, cellSender)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	// Add a stream
	sid, _ := vo.StreamIDFrom(1)
	local1, local2 := net.Pipe()
	repo.AddStream(cid, sid, local1)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// End specific stream
	payload, _ := vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16()})
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: payload}

	if err := uc.EndStream(st, cid, cell, ensureServeDown); err != nil {
		t.Fatalf("end stream error: %v", err)
	}

	// Stream should be removed
	if _, err := repo.GetStream(cid, sid); err == nil {
		t.Errorf("stream not removed")
	}

	// Connection should be closed
	local2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	if _, err := local2.Read(make([]byte, 1)); err == nil {
		t.Errorf("stream not closed")
	}

	st.Up().Close()
}

func TestHandleEndStreamUseCase_EndAllStreams(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	cellSender := service.NewCellSenderService()
	uc := usecase.NewHandleEndStreamUseCase(repo, cellSender)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	// Add multiple streams
	sid1, _ := vo.StreamIDFrom(1)
	sid2, _ := vo.StreamIDFrom(2)
	local1, _ := net.Pipe()
	local3, _ := net.Pipe()
	repo.AddStream(cid, sid1, local1)
	repo.AddStream(cid, sid2, local3)

	// Mock ensureServeDown function
	ensureServeDown := func(st *entity.ConnState) {}

	// End all streams (StreamID = 0)
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: []byte{}}

	if err := uc.EndStream(st, cid, cell, ensureServeDown); err != nil {
		t.Fatalf("end stream error: %v", err)
	}

	// Circuit should be deleted
	if _, err := repo.Find(cid); err == nil {
		t.Errorf("circuit not deleted")
	}

	// All streams should be removed
	if _, err := repo.GetStream(cid, sid1); err == nil {
		t.Errorf("stream 1 not removed")
	}
	if _, err := repo.GetStream(cid, sid2); err == nil {
		t.Errorf("stream 2 not removed")
	}
}

func TestHandleEndStreamUseCase_EndStreamWithForward(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	cellSender := service.NewCellSenderService()
	uc := usecase.NewHandleEndStreamUseCase(repo, cellSender)

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

	// End specific stream with downstream connection
	sid, _ := vo.StreamIDFrom(1)
	payload, _ := vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16()})
	cell := &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: payload}

	errCh := make(chan error, 1)
	go func() { errCh <- uc.EndStream(st, cid, cell, ensureServeDown) }()

	// Should forward cell downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdEnd {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if !serveDownCalled {
		t.Errorf("ensureServeDown not called")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("end stream error: %v", err)
	}

	st.Up().Close()
	st.Down().Close()
}
