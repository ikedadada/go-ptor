package value_object

import (
	"testing"
)

func TestCellCommand_String(t *testing.T) {
	tests := []struct {
		cmd      CellCommand
		expected string
	}{
		{CmdExtend, "EXTEND"},
		{CmdConnect, "CONNECT"},
		{CmdData, "DATA"},
		{CmdEnd, "END"},
		{CmdDestroy, "DESTROY"},
		{CmdBegin, "BEGIN"},
		{CmdBeginAck, "BEGIN_ACK"},
		{CmdCreated, "CREATED"},
	}

	for _, test := range tests {
		result := test.cmd.String()
		if result != test.expected {
			t.Errorf("CellCommand(%d).String() = %s, want %s", test.cmd, result, test.expected)
		}
	}
}

func TestCellCommand_String_Unknown(t *testing.T) {
	unknownCmd := CellCommand(0xFF)
	result := unknownCmd.String()
	expected := "UNKNOWN(255)"

	if result != expected {
		t.Errorf("Unknown CellCommand.String() = %s, want %s", result, expected)
	}
}

func TestCellCommand_IsValid(t *testing.T) {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.cmd.IsValid() {
				t.Errorf("CellCommand(%d) should be valid but IsValid() returned false", test.cmd)
			}
		})
	}
}

func TestCellCommand_IsValid_Invalid(t *testing.T) {
	tests := []struct {
		name string
		cmd  CellCommand
	}{
		{"Zero value", CellCommand(0x00)},
		{"Undefined 9", CellCommand(0x09)},
		{"Undefined 16", CellCommand(0x10)},
		{"Maximum byte", CellCommand(0xFF)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.cmd.IsValid() {
				t.Errorf("CellCommand(%d) should be invalid but IsValid() returned true", test.cmd)
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if byte(test.cmd) != test.expectedValue {
				t.Errorf("CellCommand constant value mismatch: %s = %d, want %d", test.cmd.String(), byte(test.cmd), test.expectedValue)
			}
		})
	}
}

func TestCellCommand_Coverage(t *testing.T) {
	// Test all valid commands to ensure they're covered by both String() and IsValid()
	allValidCommands := []CellCommand{
		CmdExtend,
		CmdConnect,
		CmdData,
		CmdEnd,
		CmdDestroy,
		CmdBegin,
		CmdBeginAck,
		CmdCreated,
	}

	for _, cmd := range allValidCommands {
		// Ensure String() doesn't return "UNKNOWN" for valid commands
		str := cmd.String()
		if str == "UNKNOWN("+string(rune(byte(cmd)))+")" {
			t.Errorf("Valid command %d returned UNKNOWN from String()", cmd)
		}

		// Ensure IsValid() returns true for all valid commands
		if !cmd.IsValid() {
			t.Errorf("Valid command %d returned false from IsValid()", cmd)
		}
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
