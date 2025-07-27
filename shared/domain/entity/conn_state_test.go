package entity

import (
	"net"
	"testing"
	"time"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// connStateTestConn implements net.Conn for testing ConnState operations
type connStateTestConn struct {
	closed bool
	name   string
}

func newConnStateTestConn(name string) *connStateTestConn {
	return &connStateTestConn{name: name}
}

func (c *connStateTestConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (c *connStateTestConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (c *connStateTestConn) Close() error {
	c.closed = true
	return nil
}
func (c *connStateTestConn) LocalAddr() net.Addr                { return nil }
func (c *connStateTestConn) RemoteAddr() net.Addr               { return nil }
func (c *connStateTestConn) SetDeadline(t time.Time) error      { return nil }
func (c *connStateTestConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *connStateTestConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewConnState(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := newConnStateTestConn("up")
	downConn := newConnStateTestConn("down")

	cs := NewConnState(key, nonce, upConn, downConn)

	if cs.Key() != key {
		t.Errorf("Key mismatch: got %v, want %v", cs.Key(), key)
	}
	if cs.Nonce() != nonce {
		t.Errorf("Nonce mismatch: got %v, want %v", cs.Nonce(), nonce)
	}
	if cs.Up() != upConn {
		t.Errorf("Up connection mismatch")
	}
	if cs.Down() != downConn {
		t.Errorf("Down connection mismatch")
	}

	// Check initial counter values
	beginCounter, dataCounter := cs.GetCounters()
	if beginCounter != 0 {
		t.Errorf("Initial begin counter should be 0, got %d", beginCounter)
	}
	if dataCounter != 0 {
		t.Errorf("Initial data counter should be 0, got %d", dataCounter)
	}

	// Check initial state
	if cs.IsHidden() {
		t.Error("Initial hidden state should be false")
	}
	if cs.IsServed() {
		t.Error("Initial served state should be false")
	}
}

func TestNewConnStateWithCounters(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := newConnStateTestConn("up")
	downConn := newConnStateTestConn("down")
	beginCounter := uint64(5)
	dataCounter := uint64(10)

	cs := NewConnStateWithCounters(key, nonce, upConn, downConn, beginCounter, dataCounter)

	// Check preserved counter values
	gotBeginCounter, gotDataCounter := cs.GetCounters()
	if gotBeginCounter != beginCounter {
		t.Errorf("Begin counter mismatch: got %d, want %d", gotBeginCounter, beginCounter)
	}
	if gotDataCounter != dataCounter {
		t.Errorf("Data counter mismatch: got %d, want %d", gotDataCounter, dataCounter)
	}
}

func TestConnState_BeginNonce(t *testing.T) {
	key, _ := vo.NewAESKey()
	baseNonce, _ := vo.NewNonce()
	cs := NewConnState(key, baseNonce, nil, nil)

	// Generate first nonce
	nonce1 := cs.BeginNonce()

	// Check that counter was incremented
	beginCounter, _ := cs.GetCounters()
	if beginCounter != 1 {
		t.Errorf("Begin counter should be 1 after first nonce, got %d", beginCounter)
	}

	// Generate second nonce
	nonce2 := cs.BeginNonce()

	// Check that counter was incremented again
	beginCounter, _ = cs.GetCounters()
	if beginCounter != 2 {
		t.Errorf("Begin counter should be 2 after second nonce, got %d", beginCounter)
	}

	// Nonces should be different
	if nonce1 == nonce2 {
		t.Error("Sequential nonces should be different")
	}

	// But should be based on the same base nonce
	// Check that first few bytes are the same (XOR happens in last 8 bytes)
	for i := 0; i < 4; i++ {
		if nonce1[i] != baseNonce[i] {
			t.Errorf("Nonce1 byte %d doesn't match base nonce: got %x, want %x", i, nonce1[i], baseNonce[i])
		}
		if nonce2[i] != baseNonce[i] {
			t.Errorf("Nonce2 byte %d doesn't match base nonce: got %x, want %x", i, nonce2[i], baseNonce[i])
		}
	}
}

func TestConnState_DataNonce(t *testing.T) {
	key, _ := vo.NewAESKey()
	baseNonce, _ := vo.NewNonce()
	cs := NewConnState(key, baseNonce, nil, nil)

	// Generate first nonce
	nonce1 := cs.DataNonce()

	// Check that counter was incremented
	_, dataCounter := cs.GetCounters()
	if dataCounter != 1 {
		t.Errorf("Data counter should be 1 after first nonce, got %d", dataCounter)
	}

	// Generate second nonce
	nonce2 := cs.DataNonce()

	// Check that counter was incremented again
	_, dataCounter = cs.GetCounters()
	if dataCounter != 2 {
		t.Errorf("Data counter should be 2 after second nonce, got %d", dataCounter)
	}

	// Nonces should be different
	if nonce1 == nonce2 {
		t.Error("Sequential data nonces should be different")
	}
}

func TestConnState_UpstreamDataNonce(t *testing.T) {
	key, _ := vo.NewAESKey()
	baseNonce, _ := vo.NewNonce()
	cs := NewConnState(key, baseNonce, nil, nil)

	// Generate upstream nonces
	nonce1 := cs.UpstreamDataNonce()
	nonce2 := cs.UpstreamDataNonce()

	// Nonces should be different
	if nonce1 == nonce2 {
		t.Error("Sequential upstream data nonces should be different")
	}
}

func TestConnState_Touch(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	initialTime := cs.LastUsed()

	// Wait a bit and touch
	time.Sleep(time.Millisecond)
	cs.Touch()

	newTime := cs.LastUsed()
	if !newTime.After(initialTime) {
		t.Error("Touch() should update the last used time")
	}
}

func TestConnState_HiddenFlag(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	// Initially should be false
	if cs.IsHidden() {
		t.Error("Initial hidden state should be false")
	}

	// Set to true
	cs.SetHidden(true)
	if !cs.IsHidden() {
		t.Error("Hidden state should be true after SetHidden(true)")
	}

	// Set back to false
	cs.SetHidden(false)
	if cs.IsHidden() {
		t.Error("Hidden state should be false after SetHidden(false)")
	}
}

func TestConnState_ServedFlag(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	// Initially should be false
	if cs.IsServed() {
		t.Error("Initial served state should be false")
	}

	// Mark as served
	cs.MarkServed()
	if !cs.IsServed() {
		t.Error("Served state should be true after MarkServed()")
	}
}

func TestConnState_Close(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	upConn := newConnStateTestConn("up")
	downConn := newConnStateTestConn("down")
	cs := NewConnState(key, nonce, upConn, downConn)

	// Close the connections
	cs.Close()

	// Check that both connections were closed
	if !upConn.closed {
		t.Error("Up connection should be closed")
	}
	if !downConn.closed {
		t.Error("Down connection should be closed")
	}
}

func TestConnState_CloseWithNilConnections(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	// Should not panic when closing nil connections
	cs.Close()
}

func TestConnState_GetMessageTypeNonce(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	// Test that consecutive calls to the same message type return different nonces
	beginNonce1 := cs.GetMessageTypeNonce(MessageTypeBegin)
	beginNonce2 := cs.GetMessageTypeNonce(MessageTypeBegin)
	if beginNonce1 == beginNonce2 {
		t.Error("Consecutive Begin nonces should be different")
	}

	// Test that different message types work
	dataNonce1 := cs.GetMessageTypeNonce(MessageTypeData)
	dataNonce2 := cs.GetMessageTypeNonce(MessageTypeData)
	if dataNonce1 == dataNonce2 {
		t.Error("Consecutive Data nonces should be different")
	}

	upstreamNonce1 := cs.GetMessageTypeNonce(MessageTypeUpstreamData)
	upstreamNonce2 := cs.GetMessageTypeNonce(MessageTypeUpstreamData)
	if upstreamNonce1 == upstreamNonce2 {
		t.Error("Consecutive UpstreamData nonces should be different")
	}

	// Test that Connect and Begin share the same counter mechanism
	connectNonce := cs.GetMessageTypeNonce(MessageTypeConnect)
	// Since BeginNonce was called twice already, Connect should be different from both
	if connectNonce == beginNonce1 || connectNonce == beginNonce2 {
		t.Error("Connect nonce should be different from previous Begin nonces")
	}
}

func TestConnState_IncrementCounter(t *testing.T) {
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cs := NewConnState(key, nonce, nil, nil)

	// Test incrementing different counter types
	cs.IncrementCounter(MessageTypeBegin)
	beginCounter, _ := cs.GetCounters()
	if beginCounter != 1 {
		t.Errorf("Begin counter should be 1, got %d", beginCounter)
	}

	cs.IncrementCounter(MessageTypeData)
	_, dataCounter := cs.GetCounters()
	if dataCounter != 1 {
		t.Errorf("Data counter should be 1, got %d", dataCounter)
	}

	cs.IncrementCounter(MessageTypeConnect)
	beginCounter, _ = cs.GetCounters()
	if beginCounter != 2 {
		t.Errorf("Begin counter should be 2 after Connect increment, got %d", beginCounter)
	}
}

func TestConnState_NonceUniqueness(t *testing.T) {
	key, _ := vo.NewAESKey()
	baseNonce, _ := vo.NewNonce()
	cs := NewConnState(key, baseNonce, nil, nil)

	// Generate multiple nonces and ensure they're all unique
	const numNonces = 100
	seenNonces := make(map[vo.Nonce]bool)

	for i := 0; i < numNonces; i++ {
		nonce := cs.DataNonce()
		if seenNonces[nonce] {
			t.Errorf("Duplicate nonce found at iteration %d", i)
		}
		seenNonces[nonce] = true
	}
}
