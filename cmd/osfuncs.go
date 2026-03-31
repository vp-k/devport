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
	cmdRegistrySave = registry.Save
	cmdWriteEnvFile = detect.WriteEnvFile
	cmdAllocate     = allocator.Allocate
)
