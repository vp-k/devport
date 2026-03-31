package cmd

import (
	"os"

	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
	"github.com/user01/devport/internal/resolver"
)

// Injectable OS/layer functions for testing error paths.
var (
	cmdGetwd       = os.Getwd
	cmdUserHomeDir = os.UserHomeDir
	cmdOsExit      = os.Exit

	cmdResolve      = resolver.Resolve
	cmdRegistryLoad = registry.Load
	cmdWriteEnvFile = detect.WriteEnvFile
	cmdAllocate     = allocator.Allocate

	// cmdTransaction wraps the atomic load → modify → save cycle under a
	// single write lock. Use this instead of separate Load + Save calls
	// whenever the registry is mutated, to prevent duplicate-port races.
	cmdTransaction = registry.Transaction
)
