package usecase_test

import "net"

func makeTestStabServer() (stabAddr string, err error) {
	// Setup stub server to handle net.Dial requests
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	// Get the actual address of the stub server
	stubAddr := listener.Addr().String()

	// Start stub server in background
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			conn.Close() // Close immediately for test
		}
	}()

	return stubAddr, nil
}
