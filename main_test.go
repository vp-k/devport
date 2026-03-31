package main

import (
	"os"
	"testing"
)

func TestMainInvoke(t *testing.T) {
	// Exercise main() by running "devport list" which reads the registry
	// (or creates an empty one) and prints "No projects registered." — no error.
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"devport", "list"}
	main()
}
