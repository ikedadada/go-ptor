package main

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/usecase"
)

type mockOpenUC struct {
	called bool
	in     usecase.OpenStreamInput
	out    usecase.OpenStreamOutput
	err    error
}

func (m *mockOpenUC) Handle(in usecase.OpenStreamInput) (usecase.OpenStreamOutput, error) {
	m.called = true
	m.in = in
	return m.out, m.err
}

type mockCloseUC struct {
	called bool
	in     usecase.CloseStreamInput
	err    error
}

func (m *mockCloseUC) Handle(in usecase.CloseStreamInput) (usecase.CloseStreamOutput, error) {
	m.called = true
	m.in = in
	return usecase.CloseStreamOutput{}, m.err
}

func TestHandleSOCKS(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)

	openUC := &mockOpenUC{out: usecase.OpenStreamOutput{CircuitID: "cid", StreamID: 1}}
	closeUC := &mockCloseUC{}
	done := make(chan struct{})
	go func() {
		handleSOCKS(server, "cid", openUC, closeUC)
		close(done)
	}()

	go func() {
		c, err := ln.Accept()
		if err == nil {
			c.Close()
		}
	}()

	w := bufio.NewWriter(client)
	r := bufio.NewReader(client)

	// greeting
	w.Write([]byte{5, 1, 0})
	w.Flush()
	io.ReadFull(r, make([]byte, 2))

	// connect request
	req := []byte{5, 1, 0, 1}
	req = append(req, addr.IP.To4()...)
	req = append(req, byte(addr.Port>>8), byte(addr.Port))
	w.Write(req)
	w.Flush()
	io.ReadFull(r, make([]byte, 10))

	client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("handleSOCKS did not exit")
	}

	if !openUC.called {
		t.Errorf("open UC not called")
	}
	if !closeUC.called {
		t.Errorf("close UC not called")
	}
}
