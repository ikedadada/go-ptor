package repository

import (
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// relayConnStateTestConn implements net.Conn for testing
type relayConnStateTestConn struct {
	closed bool
}

func (c *relayConnStateTestConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *relayConnStateTestConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (c *relayConnStateTestConn) Close() error {
	c.closed = true
	return nil
}

func (c *relayConnStateTestConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
}

func (c *relayConnStateTestConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9090}
}

func (c *relayConnStateTestConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *relayConnStateTestConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *relayConnStateTestConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestNewConnStateRepository(t *testing.T) {
	ttl := time.Minute
	repo := NewConnStateRepository(ttl)

	if repo == nil {
		t.Error("NewConnStateRepository should return a non-nil repository")
	}

	// Verify it implements ConnStateRepository interface
	var _ repository.ConnStateRepository = repo
}

func TestConnStateRepository_Add(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Create a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify we can find the added state
	found, err := repo.Find(circuitID)
	if err != nil {
		t.Fatalf("Find failed after Add: %v", err)
	}
	if found != state {
		t.Error("Found state does not match added state")
	}
}

func TestConnStateRepository_Find_NotFound(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	_, err := repo.Find(circuitID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestConnStateRepository_Delete(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Create and add a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Delete the state
	err = repo.Delete(circuitID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's no longer found
	_, err = repo.Find(circuitID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestConnStateRepository_Delete_WithStreams(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()

	// Create and add a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Add a stream
	testConn := &relayConnStateTestConn{}
	err = repo.AddStream(circuitID, streamID, testConn)
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	// Delete the circuit (should close streams)
	err = repo.Delete(circuitID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify stream connection was closed
	if !testConn.closed {
		t.Error("Stream connection should have been closed when circuit was deleted")
	}

	// Verify stream is no longer found
	_, err = repo.GetStream(circuitID, streamID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound for stream after circuit delete, got: %v", err)
	}
}

func TestConnStateRepository_AddStream(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()
	testConn := &relayConnStateTestConn{}

	err := repo.AddStream(circuitID, streamID, testConn)
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	// Verify we can get the stream
	conn, err := repo.GetStream(circuitID, streamID)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}
	if conn != testConn {
		t.Error("Retrieved stream does not match added stream")
	}
}

func TestConnStateRepository_GetStream_NotFound(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()

	// Test non-existent circuit
	_, err := repo.GetStream(circuitID, streamID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent circuit, got: %v", err)
	}

	// Add a circuit but not the stream
	otherStreamID := vo.NewStreamIDAuto()
	err = repo.AddStream(circuitID, otherStreamID, &relayConnStateTestConn{})
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	// Test non-existent stream in existing circuit
	_, err = repo.GetStream(circuitID, streamID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent stream, got: %v", err)
	}
}

func TestConnStateRepository_RemoveStream(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()
	testConn := &relayConnStateTestConn{}

	// Add a stream
	err := repo.AddStream(circuitID, streamID, testConn)
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	// Remove the stream
	err = repo.RemoveStream(circuitID, streamID)
	if err != nil {
		t.Fatalf("RemoveStream failed: %v", err)
	}

	// Verify connection was closed
	if !testConn.closed {
		t.Error("Stream connection should have been closed when removed")
	}

	// Verify stream is no longer found
	_, err = repo.GetStream(circuitID, streamID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound after RemoveStream, got: %v", err)
	}
}

func TestConnStateRepository_RemoveStream_NonExistent(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()

	// Remove non-existent stream should not error
	err := repo.RemoveStream(circuitID, streamID)
	if err != nil {
		t.Errorf("RemoveStream of non-existent stream should not error, got: %v", err)
	}
}

func TestConnStateRepository_RemoveStream_CleansUpEmptyCircuit(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()
	testConn := &relayConnStateTestConn{}

	// Add a single stream
	err := repo.AddStream(circuitID, streamID, testConn)
	if err != nil {
		t.Fatalf("AddStream failed: %v", err)
	}

	// Remove the stream (should clean up empty circuit map)
	err = repo.RemoveStream(circuitID, streamID)
	if err != nil {
		t.Fatalf("RemoveStream failed: %v", err)
	}

	// Verify circuit is completely removed from streams map
	_, err = repo.GetStream(circuitID, streamID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected ErrNotFound after removing last stream, got: %v", err)
	}
}

func TestConnStateRepository_DestroyAllStreams(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Add multiple streams
	testConns := []*relayConnStateTestConn{
		{},
		{},
		{},
	}

	streamIDs := make([]vo.StreamID, len(testConns))
	for i, conn := range testConns {
		streamIDs[i] = vo.NewStreamIDAuto()
		err := repo.AddStream(circuitID, streamIDs[i], conn)
		if err != nil {
			t.Fatalf("AddStream %d failed: %v", i, err)
		}
	}

	// Destroy all streams
	repo.DestroyAllStreams(circuitID)

	// Verify all connections were closed
	for i, conn := range testConns {
		if !conn.closed {
			t.Errorf("Stream %d connection should have been closed", i)
		}
	}

	// Verify no streams can be found
	for i := range testConns {
		_, err := repo.GetStream(circuitID, streamIDs[i])
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound for stream %d after DestroyAllStreams, got: %v", i, err)
		}
	}
}

func TestConnStateRepository_DestroyAllStreams_NonExistent(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Destroying streams for non-existent circuit should not panic
	repo.DestroyAllStreams(circuitID)
}

func TestConnStateRepository_MultipleStreamsPerCircuit(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Add multiple streams
	numStreams := 5
	testConns := make([]*relayConnStateTestConn, numStreams)

	streamIDs := make([]vo.StreamID, numStreams)
	for i := 0; i < numStreams; i++ {
		testConns[i] = &relayConnStateTestConn{}
		streamIDs[i] = vo.NewStreamIDAuto()
		err := repo.AddStream(circuitID, streamIDs[i], testConns[i])
		if err != nil {
			t.Fatalf("AddStream %d failed: %v", i, err)
		}
	}

	// Verify all streams can be retrieved
	for i := 0; i < numStreams; i++ {
		conn, err := repo.GetStream(circuitID, streamIDs[i])
		if err != nil {
			t.Fatalf("GetStream %d failed: %v", i, err)
		}
		if conn != testConns[i] {
			t.Errorf("Stream %d connection mismatch", i)
		}
	}

	// Remove one stream (remove the third stream)
	removedStreamID := streamIDs[2]
	err := repo.RemoveStream(circuitID, removedStreamID)
	if err != nil {
		t.Fatalf("RemoveStream failed: %v", err)
	}

	// Verify removed stream is gone but others remain
	_, err = repo.GetStream(circuitID, removedStreamID)
	if err != repository.ErrNotFound {
		t.Error("Removed stream should not be found")
	}

	// Verify other streams still exist
	for i := 0; i < numStreams; i++ {
		if i == 2 { // Skip the removed stream (index 2)
			continue
		}
		_, err := repo.GetStream(circuitID, streamIDs[i])
		if err != nil {
			t.Errorf("Stream %d should still exist after removing stream: %v", i, err)
		}
	}
}

func TestConnStateRepository_GarbageCollection(t *testing.T) {
	// Use a TTL that's longer than minimum GC interval (1 second)
	ttl := 2 * time.Second
	repo := NewConnStateRepository(ttl)
	circuitID := vo.NewCircuitID()

	// Create and add a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify state exists
	_, err = repo.Find(circuitID)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	// Wait for TTL to expire plus some buffer for GC to run
	time.Sleep(ttl + time.Second*2)

	// Verify state was garbage collected
	_, err = repo.Find(circuitID)
	if err != repository.ErrNotFound {
		t.Errorf("Expected state to be garbage collected, got: %v", err)
	}
}

func TestConnStateRepository_TouchUpdatesLastUsed(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Create and add a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	initialTime := state.LastUsed()

	// Wait a bit then access the state (should call Touch)
	time.Sleep(10 * time.Millisecond)
	_, err = repo.Find(circuitID)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	updatedTime := state.LastUsed()
	if !updatedTime.After(initialTime) {
		t.Error("LastUsed should be updated after Find operation")
	}
}

func TestConnStateRepository_ConcurrentAccess(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()

	// Create and add a test connection state
	key := vo.AESKey{}
	copy(key[:], make([]byte, 32))
	baseNonce := vo.Nonce{}
	copy(baseNonce[:], make([]byte, 12))
	state := entity.NewConnState(key, baseNonce, nil, nil)

	err := repo.Add(circuitID, state)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Test concurrent access doesn't cause race conditions
	done := make(chan bool, 2)

	// Goroutine 1: Find operations
	go func() {
		for i := 0; i < 100; i++ {
			repo.Find(circuitID)
		}
		done <- true
	}()

	// Goroutine 2: Stream operations
	go func() {
		for i := 0; i < 100; i++ {
			streamID := vo.NewStreamIDAuto()
			conn := &relayConnStateTestConn{}
			repo.AddStream(circuitID, streamID, conn)
			repo.GetStream(circuitID, streamID)
			repo.RemoveStream(circuitID, streamID)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify repository is still functional
	_, err = repo.Find(circuitID)
	if err != nil {
		t.Errorf("Repository should still be functional after concurrent access: %v", err)
	}
}

func TestConnStateRepository_NilConnHandling(t *testing.T) {
	repo := NewConnStateRepository(time.Minute)
	circuitID := vo.NewCircuitID()
	streamID := vo.NewStreamIDAuto()

	// Add a nil connection
	err := repo.AddStream(circuitID, streamID, nil)
	if err != nil {
		t.Fatalf("AddStream with nil connection failed: %v", err)
	}

	// Verify we can get the nil connection
	conn, err := repo.GetStream(circuitID, streamID)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}
	if conn != nil {
		t.Error("Expected nil connection")
	}

	// Remove the nil connection (should not panic)
	err = repo.RemoveStream(circuitID, streamID)
	if err != nil {
		t.Errorf("RemoveStream with nil connection failed: %v", err)
	}
}
