package service

import (
	"net"
	"sync"
	"testing"
	"time"
)

// streamManagerTestConn implements net.Conn for testing stream management operations
type streamManagerTestConn struct {
	id     uint16
	closed bool
	mu     sync.Mutex
}

func newStreamManagerTestConn(id uint16) *streamManagerTestConn {
	return &streamManagerTestConn{id: id}
}

func (m *streamManagerTestConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *streamManagerTestConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *streamManagerTestConn) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *streamManagerTestConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *streamManagerTestConn) LocalAddr() net.Addr                { return nil }
func (m *streamManagerTestConn) RemoteAddr() net.Addr               { return nil }
func (m *streamManagerTestConn) SetDeadline(t time.Time) error      { return nil }
func (m *streamManagerTestConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *streamManagerTestConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewStreamManagerService(t *testing.T) {
	sm := NewStreamManagerService()
	if sm == nil {
		t.Fatal("NewStreamManagerService should not return nil")
	}
}

func TestStreamManagerService_Add_Get(t *testing.T) {
	sm := NewStreamManagerService()
	conn := newStreamManagerTestConn(1)

	// Add connection
	sm.Add(1, conn)

	// Get connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Error("Get should return true for existing connection")
	}

	if retrievedConn != conn {
		t.Error("Retrieved connection should be the same as added")
	}
}

func TestStreamManagerService_Get_NonExistent(t *testing.T) {
	sm := NewStreamManagerService()

	// Try to get non-existent connection
	conn, ok := sm.Get(999)
	if ok {
		t.Error("Get should return false for non-existent connection")
	}

	if conn != nil {
		t.Error("Connection should be nil for non-existent ID")
	}
}

func TestStreamManagerService_Remove(t *testing.T) {
	sm := NewStreamManagerService()
	conn := newStreamManagerTestConn(1)

	// Add connection
	sm.Add(1, conn)

	// Verify it exists
	_, ok := sm.Get(1)
	if !ok {
		t.Fatal("Connection should exist before removal")
	}

	// Remove connection
	sm.Remove(1)

	// Verify connection is closed
	if !conn.IsClosed() {
		t.Error("Connection should be closed after removal")
	}

	// Verify it no longer exists
	_, ok = sm.Get(1)
	if ok {
		t.Error("Connection should not exist after removal")
	}
}

func TestStreamManagerService_Remove_NonExistent(t *testing.T) {
	sm := NewStreamManagerService()

	// Remove non-existent connection (should not panic)
	sm.Remove(999)
}

func TestStreamManagerService_CloseAll(t *testing.T) {
	sm := NewStreamManagerService()

	// Add multiple connections
	conn1 := newStreamManagerTestConn(1)
	conn2 := newStreamManagerTestConn(2)
	conn3 := newStreamManagerTestConn(3)

	sm.Add(1, conn1)
	sm.Add(2, conn2)
	sm.Add(3, conn3)

	// Verify connections exist
	_, ok1 := sm.Get(1)
	_, ok2 := sm.Get(2)
	_, ok3 := sm.Get(3)
	if !ok1 || !ok2 || !ok3 {
		t.Fatal("All connections should exist before CloseAll")
	}

	// Close all connections
	sm.CloseAll()

	// Verify all connections are closed
	if !conn1.IsClosed() {
		t.Error("Connection 1 should be closed")
	}
	if !conn2.IsClosed() {
		t.Error("Connection 2 should be closed")
	}
	if !conn3.IsClosed() {
		t.Error("Connection 3 should be closed")
	}

	// Verify no connections exist
	_, ok1 = sm.Get(1)
	_, ok2 = sm.Get(2)
	_, ok3 = sm.Get(3)
	if ok1 || ok2 || ok3 {
		t.Error("No connections should exist after CloseAll")
	}
}

func TestStreamManagerService_CloseAll_Empty(t *testing.T) {
	sm := NewStreamManagerService()

	// Close all on empty manager (should not panic)
	sm.CloseAll()
}

func TestStreamManagerService_Add_Overwrite(t *testing.T) {
	sm := NewStreamManagerService()

	conn1 := newStreamManagerTestConn(1)
	conn2 := newStreamManagerTestConn(2)

	// Add first connection
	sm.Add(1, conn1)

	// Overwrite with second connection
	sm.Add(1, conn2)

	// Get connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Fatal("Connection should exist")
	}

	// Should get the second connection
	if retrievedConn != conn2 {
		t.Error("Should get the overwritten connection")
	}
}

func TestStreamManagerService_ConcurrentAccess(t *testing.T) {
	sm := NewStreamManagerService()
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup

	// Start multiple goroutines that add/get/remove connections
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				streamID := uint16(workerID*numOperations + j)
				conn := newStreamManagerTestConn(streamID)

				// Add connection
				sm.Add(streamID, conn)

				// Get connection
				retrievedConn, ok := sm.Get(streamID)
				if !ok {
					t.Errorf("Connection %d should exist", streamID)
					continue
				}

				if retrievedConn != conn {
					t.Errorf("Retrieved connection %d should match added connection", streamID)
				}

				// Remove connection
				sm.Remove(streamID)

				// Verify removal
				_, ok = sm.Get(streamID)
				if ok {
					t.Errorf("Connection %d should not exist after removal", streamID)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestStreamManagerService_ConcurrentCloseAll(t *testing.T) {
	sm := NewStreamManagerService()
	numConnections := 100

	// Add many connections
	var connections []*streamManagerTestConn
	for i := 0; i < numConnections; i++ {
		conn := newStreamManagerTestConn(uint16(i))
		connections = append(connections, conn)
		sm.Add(uint16(i), conn)
	}

	var wg sync.WaitGroup

	// Start multiple goroutines that call CloseAll simultaneously
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.CloseAll()
		}()
	}

	wg.Wait()

	// Verify all connections are closed
	for i, conn := range connections {
		if !conn.IsClosed() {
			t.Errorf("Connection %d should be closed", i)
		}
	}

	// Verify no connections exist
	for i := 0; i < numConnections; i++ {
		_, ok := sm.Get(uint16(i))
		if ok {
			t.Errorf("Connection %d should not exist after CloseAll", i)
		}
	}
}

func TestStreamManagerService_AddAfterCloseAll(t *testing.T) {
	sm := NewStreamManagerService()

	// Add a connection
	conn1 := newStreamManagerTestConn(1)
	sm.Add(1, conn1)

	// Close all
	sm.CloseAll()

	// Add new connection with same ID
	conn2 := newStreamManagerTestConn(2)
	sm.Add(1, conn2)

	// Should get the new connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Fatal("New connection should exist")
	}

	if retrievedConn != conn2 {
		t.Error("Should get the new connection after CloseAll")
	}

	// Old connection should still be closed
	if !conn1.IsClosed() {
		t.Error("Old connection should remain closed")
	}
}

func TestStreamManagerService_MaxStreamID(t *testing.T) {
	sm := NewStreamManagerService()

	// Test with maximum uint16 value
	maxID := uint16(65535)
	conn := newStreamManagerTestConn(maxID)

	sm.Add(maxID, conn)

	retrievedConn, ok := sm.Get(maxID)
	if !ok {
		t.Error("Should handle maximum stream ID")
	}

	if retrievedConn != conn {
		t.Error("Should retrieve correct connection for maximum stream ID")
	}

	sm.Remove(maxID)

	_, ok = sm.Get(maxID)
	if ok {
		t.Error("Connection should not exist after removal")
	}
}

func TestStreamManagerService_ZeroStreamID(t *testing.T) {
	sm := NewStreamManagerService()

	// Test with zero stream ID
	conn := newStreamManagerTestConn(0)

	sm.Add(0, conn)

	retrievedConn, ok := sm.Get(0)
	if !ok {
		t.Error("Should handle zero stream ID")
	}

	if retrievedConn != conn {
		t.Error("Should retrieve correct connection for zero stream ID")
	}

	sm.Remove(0)

	_, ok = sm.Get(0)
	if ok {
		t.Error("Connection should not exist after removal")
	}
}
