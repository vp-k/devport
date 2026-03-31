package allocator

import (
	"net"
	"testing"
)

func TestProbePortAvailable(t *testing.T) {
	// Use port 0 to let the OS assign a free port, then check that port is available
	// before we bind it ourselves.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // release it

	// Now it should be available.
	if !ProbePort(port) {
		t.Errorf("port %d should be available but ProbePort returned false", port)
	}
}

func TestProbePortInUse(t *testing.T) {
	// Bind a port and verify ProbePort reports it as in use.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if ProbePort(port) {
		t.Errorf("port %d is in use but ProbePort returned true", port)
	}
}
