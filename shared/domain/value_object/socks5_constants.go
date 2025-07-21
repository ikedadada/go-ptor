package value_object

// SOCKS5 protocol constants
const (
	SOCKS5Version      = 5
	SOCKS5MethodNoAuth = 0
	SOCKS5CmdConnect   = 1

	// Address types
	SOCKS5AddrIPv4   = 1
	SOCKS5AddrDomain = 3
	SOCKS5AddrIPv6   = 4

	// Response codes
	SOCKS5RespSuccess      = 0
	SOCKS5RespGeneralError = 1
	SOCKS5RespHostUnreach  = 4
)

// SOCKS5 response templates
var (
	SOCKS5HandshakeResp   = []byte{SOCKS5Version, SOCKS5MethodNoAuth}
	SOCKS5SuccessResp     = []byte{SOCKS5Version, SOCKS5RespSuccess, 0, 1, 0, 0, 0, 0, 0, 0}
	SOCKS5ErrorResp       = []byte{SOCKS5Version, SOCKS5RespGeneralError, 0, 1, 0, 0, 0, 0, 0, 0}
	SOCKS5HostUnreachResp = []byte{SOCKS5Version, SOCKS5RespHostUnreach, 0, 1, 0, 0, 0, 0, 0, 0}
)
