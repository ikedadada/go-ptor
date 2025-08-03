package value_object

import (
	"testing"
)

func TestCellCommand_String(t *testing.T) {
	tests := []struct {
		name     string
		cmd      CellCommand
		expected string
	}{
		{"EXTEND", CmdExtend, "EXTEND"},
		{"CONNECT", CmdConnect, "CONNECT"},
		{"DATA", CmdData, "DATA"},
		{"END", CmdEnd, "END"},
		{"DESTROY", CmdDestroy, "DESTROY"},
		{"BEGIN", CmdBegin, "BEGIN"},
		{"BEGIN_ACK", CmdBeginAck, "BEGIN_ACK"},
		{"CREATED", CmdCreated, "CREATED"},
		{"UNKNOWN_255", CellCommand(0xFF), "UNKNOWN(255)"},
		{"UNKNOWN_0", CellCommand(0x00), "UNKNOWN(0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.String()
			if result != tt.expected {
				t.Errorf("CellCommand(%d).String() = %s, want %s", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestCellCommand_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		cmd         CellCommand
		expectValid bool
	}{
		{"CmdExtend", CmdExtend, true},
		{"CmdConnect", CmdConnect, true},
		{"CmdData", CmdData, true},
		{"CmdEnd", CmdEnd, true},
		{"CmdDestroy", CmdDestroy, true},
		{"CmdBegin", CmdBegin, true},
		{"CmdBeginAck", CmdBeginAck, true},
		{"CmdCreated", CmdCreated, true},
		{"Zero value", CellCommand(0x00), false},
		{"Undefined 9", CellCommand(0x09), false},
		{"Undefined 16", CellCommand(0x10), false},
		{"Maximum byte", CellCommand(0xFF), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cmd.IsValid()
			if isValid != tt.expectValid {
				t.Errorf("CellCommand(%d).IsValid() = %v, want %v", tt.cmd, isValid, tt.expectValid)
			}
		})
	}
}

func TestCellCommand_Constants(t *testing.T) {
	tests := []struct {
		name          string
		cmd           CellCommand
		expectedValue byte
	}{
		{"CmdExtend", CmdExtend, 0x01},
		{"CmdConnect", CmdConnect, 0x02},
		{"CmdData", CmdData, 0x03},
		{"CmdEnd", CmdEnd, 0x04},
		{"CmdDestroy", CmdDestroy, 0x05},
		{"CmdBegin", CmdBegin, 0x06},
		{"CmdBeginAck", CmdBeginAck, 0x07},
		{"CmdCreated", CmdCreated, 0x08},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if byte(tt.cmd) != tt.expectedValue {
				t.Errorf("CellCommand constant value mismatch: %s = %d, want %d", tt.cmd.String(), byte(tt.cmd), tt.expectedValue)
			}
		})
	}
}

func TestCellCommand_Coverage(t *testing.T) {
	tests := []struct {
		name string
		cmd  CellCommand
	}{
		{"CmdExtend", CmdExtend},
		{"CmdConnect", CmdConnect},
		{"CmdData", CmdData},
		{"CmdEnd", CmdEnd},
		{"CmdDestroy", CmdDestroy},
		{"CmdBegin", CmdBegin},
		{"CmdBeginAck", CmdBeginAck},
		{"CmdCreated", CmdCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure String() doesn't return "UNKNOWN" for valid commands
			str := tt.cmd.String()
			if str == "UNKNOWN("+string(rune(byte(tt.cmd)))+")" {
				t.Errorf("Valid command %d returned UNKNOWN from String()", tt.cmd)
			}

			// Ensure IsValid() returns true for all valid commands
			if !tt.cmd.IsValid() {
				t.Errorf("Valid command %d returned false from IsValid()", tt.cmd)
			}
		})
	}
}

func TestCellCommand_TypeSafety(t *testing.T) {
	// Test that CellCommand is properly typed as byte
	var cmd CellCommand = CmdData
	var b byte = byte(cmd)

	if b != 0x03 {
		t.Errorf("CellCommand type conversion failed: got %d, want 3", b)
	}

	// Test conversion back
	cmd2 := CellCommand(b)
	if cmd2 != CmdData {
		t.Errorf("Byte to CellCommand conversion failed: got %d, want %d", cmd2, CmdData)
	}
}
