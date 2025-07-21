package value_object

import "fmt"

// CellCommand represents the type of cell command in the Tor protocol
type CellCommand byte

const (
	// Cell command types
	CmdExtend   CellCommand = 0x01
	CmdConnect  CellCommand = 0x02
	CmdData     CellCommand = 0x03
	CmdEnd      CellCommand = 0x04
	CmdDestroy  CellCommand = 0x05
	CmdBegin    CellCommand = 0x06
	CmdBeginAck CellCommand = 0x07
	CmdCreated  CellCommand = 0x08
)

// String returns the string representation of the cell command
func (c CellCommand) String() string {
	switch c {
	case CmdExtend:
		return "EXTEND"
	case CmdConnect:
		return "CONNECT"
	case CmdData:
		return "DATA"
	case CmdEnd:
		return "END"
	case CmdDestroy:
		return "DESTROY"
	case CmdBegin:
		return "BEGIN"
	case CmdBeginAck:
		return "BEGIN_ACK"
	case CmdCreated:
		return "CREATED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", byte(c))
	}
}

// IsValid checks if the command is a valid cell command
func (c CellCommand) IsValid() bool {
	switch c {
	case CmdExtend, CmdConnect, CmdData, CmdEnd, CmdDestroy, CmdBegin, CmdBeginAck, CmdCreated:
		return true
	default:
		return false
	}
}
