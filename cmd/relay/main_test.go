package main

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ovechkin-dm/mockio/v2/matchers"
	. "github.com/ovechkin-dm/mockio/v2/mock"
	"ikedadada/go-ptor/shared/domain/aggregate"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestDefaultTTLFromEnv(t *testing.T) {
	os.Setenv("PTOR_TTL_SECONDS", "10")
	defer os.Unsetenv("PTOR_TTL_SECONDS")
	if got := defaultTTL(); got != 10*time.Second {
		t.Fatalf("expected 10s, got %v", got)
	}
}

// Helper struct to track connection state for TCP dialer tests
type relayRecordConnState struct {
	bytes.Buffer
}

// Helper function to create record connection mock for relay tests
func createRelayRecordMockConnection(ctrl *matchers.MockController) (net.Conn, *relayRecordConnState) {
	mockConn := Mock[net.Conn](ctrl)
	state := &relayRecordConnState{}

	// Set up Read behavior to read from buffer
	WhenDouble(mockConn.Read(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		b := args[0].([]byte)
		return state.Buffer.Read(b)
	})

	// Set up Write behavior to write to buffer
	WhenDouble(mockConn.Write(Any[[]byte]())).ThenAnswer(func(args []any) (int, error) {
		b := args[0].([]byte)
		return state.Buffer.Write(b)
	})

	// Set up other methods with default behaviors
	WhenSingle(mockConn.Close()).ThenReturn(nil)
	WhenSingle(mockConn.LocalAddr()).ThenReturn(nil)
	WhenSingle(mockConn.RemoteAddr()).ThenReturn(nil)
	WhenSingle(mockConn.SetDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetReadDeadline(Any[time.Time]())).ThenReturn(nil)
	WhenSingle(mockConn.SetWriteDeadline(Any[time.Time]())).ThenReturn(nil)

	return mockConn, state
}

func TestSendCellWritesFixedPacket(t *testing.T) {
	ctrl := NewMockController(t)
	conn, connState := createRelayRecordMockConnection(ctrl)
	d := service.NewTCPCircuitBuildService()
	cid := vo.NewCircuitID()
	payload := []byte("hello")
	streamID, _ := vo.StreamIDFrom(0)
	cell, err := aggregate.NewRelayCell(vo.CmdExtend, cid, streamID, payload)
	if err != nil {
		t.Fatalf("NewRelayCell error: %v", err)
	}

	if err := d.SendExtendCell(conn, cell); err != nil {
		t.Fatalf("SendExtendCell error: %v", err)
	}

	if connState.Len() != 16+entity.MaxCellSize {
		t.Fatalf("expected %d bytes, got %d", 16+entity.MaxCellSize, connState.Len())
	}
}
