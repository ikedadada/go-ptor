package handler

import (
	"bufio"
	"io"
	"net"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/usecase"
)

// --- mocks ---------------------------------------------------------------

type mockOpen struct{}

func (mockOpen) Handle(usecase.OpenStreamInput) (usecase.OpenStreamOutput, error) {
	return usecase.OpenStreamOutput{StreamID: 1}, nil
}

type mockClose struct{}

func (mockClose) Handle(usecase.CloseStreamInput) (usecase.CloseStreamOutput, error) {
	return usecase.CloseStreamOutput{}, nil
}

type mockSend struct{}

func (mockSend) Handle(usecase.SendDataInput) (usecase.SendDataOutput, error) {
	return usecase.SendDataOutput{}, nil
}

type mockEnd struct{}

func (mockEnd) Handle(usecase.HandleEndInput) (usecase.HandleEndOutput, error) {
	return usecase.HandleEndOutput{}, nil
}

// -------------------------------------------------------------------------

func TestClientHandler_StartSOCKS(t *testing.T) {
	h := NewClientHandler(entity.Directory{}, "cid", mockOpen{}, mockClose{}, mockSend{}, mockEnd{})
	ln, err := h.StartSOCKS("127.0.0.1:0")
	if err != nil {
		t.Fatalf("start socks: %v", err)
	}
	defer ln.Close()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	if _, err := w.Write([]byte{5, 1, 0}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	w.Flush()
	if _, err := io.ReadFull(r, make([]byte, 2)); err != nil {
		t.Fatalf("read hello resp: %v", err)
	}

	if _, err := w.Write([]byte{5, 1, 0, 1, 127, 0, 0, 1, 0, 80}); err != nil {
		t.Fatalf("write req: %v", err)
	}
	w.Flush()
	if _, err := io.ReadFull(r, make([]byte, 10)); err != nil {
		t.Fatalf("read resp: %v", err)
	}
}
