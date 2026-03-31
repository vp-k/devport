package allocator

import (
	"fmt"
	"net"
)

// ProbePort returns true if the given TCP port is available on 127.0.0.1.
// It works by attempting to bind the port; if binding succeeds the port is free.
func ProbePort(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
