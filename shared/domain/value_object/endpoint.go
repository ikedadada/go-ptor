package value_object

import (
	"fmt"
	"net"
)

// Endpoint は "host:port" を表す値オブジェクト
type Endpoint struct {
	host string
	port uint16
}

func NewEndpoint(host string, port uint16) (Endpoint, error) {
	if port == 0 {
		return Endpoint{}, fmt.Errorf("invalid port: %d", port)
	}
	if ip := net.ParseIP(host); ip == nil && host == "" {
		return Endpoint{}, fmt.Errorf("invalid host")
	}
	return Endpoint{host, port}, nil
}

func (e Endpoint) String() string { return fmt.Sprintf("%s:%d", e.host, e.port) }
