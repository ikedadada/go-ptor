package service

import (
	"net"
	"sync"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

func TestNewStreamManagerService(t *testing.T) {
	sm := NewStreamManagerService()
	if sm == nil {
		t.Fatal("NewStreamManagerService should not return nil")
	}
}

func TestStreamManagerService_Add_Get(t *testing.T) {
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()
	mockConn := Mock[net.Conn](ctrl)

	// Add connection
	sm.Add(1, mockConn)

	// Get connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Error("Get should return true for existing connection")
	}

	if retrievedConn != mockConn {
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
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()
	mockConn := Mock[net.Conn](ctrl)

	// Mock Close() to return nil
	WhenSingle(mockConn.Close()).ThenReturn(nil)

	// Add connection
	sm.Add(1, mockConn)

	// Verify it exists
	_, ok := sm.Get(1)
	if !ok {
		t.Fatal("Connection should exist before removal")
	}

	// Remove connection
	sm.Remove(1)

	// Verify Close() was called
	Verify(mockConn, Times(1)).Close()

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
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()

	// Add multiple connections
	mockConn1 := Mock[net.Conn](ctrl)
	mockConn2 := Mock[net.Conn](ctrl)
	mockConn3 := Mock[net.Conn](ctrl)

	// Mock Close() for all connections
	WhenSingle(mockConn1.Close()).ThenReturn(nil)
	WhenSingle(mockConn2.Close()).ThenReturn(nil)
	WhenSingle(mockConn3.Close()).ThenReturn(nil)

	sm.Add(1, mockConn1)
	sm.Add(2, mockConn2)
	sm.Add(3, mockConn3)

	// Verify connections exist
	_, ok1 := sm.Get(1)
	_, ok2 := sm.Get(2)
	_, ok3 := sm.Get(3)
	if !ok1 || !ok2 || !ok3 {
		t.Fatal("All connections should exist before CloseAll")
	}

	// Close all connections
	sm.CloseAll()

	// Verify all connections were closed
	Verify(mockConn1, Times(1)).Close()
	Verify(mockConn2, Times(1)).Close()
	Verify(mockConn3, Times(1)).Close()

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
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()

	mockConn1 := Mock[net.Conn](ctrl)
	mockConn2 := Mock[net.Conn](ctrl)

	// Add first connection
	sm.Add(1, mockConn1)

	// Overwrite with second connection
	sm.Add(1, mockConn2)

	// Get connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Fatal("Connection should exist")
	}

	// Should get the second connection
	if retrievedConn != mockConn2 {
		t.Error("Should get the overwritten connection")
	}
}

func TestStreamManagerService_ConcurrentAccess(t *testing.T) {
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()
	numGoroutines := 10
	numOperations := 10 // Reduced from 100 with pre-created mocks to avoid Mockio race conditions

	// Pre-create mocks to avoid concurrent mock creation
	mocks := make(map[vo.StreamID]net.Conn)
	for i := 0; i < numGoroutines; i++ {
		for j := 1; j < numOperations; j++ {
			streamID, err := vo.StreamIDFrom(uint16(i*numOperations + j))
			if err != nil {
				continue
			}
			mockConn := Mock[net.Conn](ctrl)
			WhenSingle(mockConn.Close()).ThenReturn(nil)
			mocks[streamID] = mockConn
		}
	}

	var wg sync.WaitGroup

	// Start multiple goroutines that add/get/remove connections
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 1; j < numOperations; j++ {
				streamID, err := vo.StreamIDFrom(uint16(workerID*numOperations + j))
				if err != nil {
					t.Errorf("Failed to create StreamID: %v", err)
					continue
				}
				mockConn := mocks[streamID]

				// Add connection
				sm.Add(streamID, mockConn)

				// Get connection
				retrievedConn, ok := sm.Get(streamID)
				if !ok {
					t.Errorf("Connection %d should exist", streamID)
					continue
				}

				if retrievedConn != mockConn {
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
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()
	numConnections := 100

	// Add many connections
	for i := 1; i < numConnections; i++ {
		sid, err := vo.StreamIDFrom(uint16(i))
		if err != nil {
			t.Fatalf("Failed to create StreamID: %v", err)
		}
		mockConn := Mock[net.Conn](ctrl)
		// Mock Close() for each connection (allowing multiple calls)
		WhenSingle(mockConn.Close()).ThenReturn(nil)
		sm.Add(sid, mockConn)
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

	// Verify no connections exist
	for i := 1; i < numConnections; i++ {
		sid, err := vo.StreamIDFrom(uint16(i))
		if err != nil {
			t.Fatalf("Failed to create StreamID: %v", err)
		}
		_, ok := sm.Get(sid)
		if ok {
			t.Errorf("Connection %d should not exist after CloseAll", i)
		}
	}
}

func TestStreamManagerService_AddAfterCloseAll(t *testing.T) {
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()

	// Add a connection
	mockConn1 := Mock[net.Conn](ctrl)
	WhenSingle(mockConn1.Close()).ThenReturn(nil)
	sm.Add(1, mockConn1)

	// Close all
	sm.CloseAll()

	// Add new connection with same ID
	mockConn2 := Mock[net.Conn](ctrl)
	sm.Add(1, mockConn2)

	// Should get the new connection
	retrievedConn, ok := sm.Get(1)
	if !ok {
		t.Fatal("New connection should exist")
	}

	if retrievedConn != mockConn2 {
		t.Error("Should get the new connection after CloseAll")
	}

	// Verify the old connection was closed
	Verify(mockConn1, Times(1)).Close()
}

func TestStreamManagerService_MaxStreamID(t *testing.T) {
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()

	// Test with maximum uint16 value
	maxID := uint16(65535)
	sid := vo.StreamID(maxID)
	mockConn := Mock[net.Conn](ctrl)
	WhenSingle(mockConn.Close()).ThenReturn(nil)

	sm.Add(sid, mockConn)

	retrievedConn, ok := sm.Get(sid)
	if !ok {
		t.Error("Should handle maximum stream ID")
	}

	if retrievedConn != mockConn {
		t.Error("Should retrieve correct connection for maximum stream ID")
	}

	sm.Remove(sid)

	_, ok = sm.Get(sid)
	if ok {
		t.Error("Connection should not exist after removal")
	}

	Verify(mockConn, Times(1)).Close()
}

func TestStreamManagerService_ZeroStreamID(t *testing.T) {
	ctrl := NewMockController(t)
	sm := NewStreamManagerService()

	// Test with zero stream ID
	mockConn := Mock[net.Conn](ctrl)
	WhenSingle(mockConn.Close()).ThenReturn(nil)

	sm.Add(0, mockConn)

	retrievedConn, ok := sm.Get(0)
	if !ok {
		t.Error("Should handle zero stream ID")
	}

	if retrievedConn != mockConn {
		t.Error("Should retrieve correct connection for zero stream ID")
	}

	sm.Remove(0)

	_, ok = sm.Get(0)
	if ok {
		t.Error("Connection should not exist after removal")
	}

	Verify(mockConn, Times(1)).Close()
}
