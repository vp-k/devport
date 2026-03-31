package allocator

import (
	"fmt"

	"github.com/user01/devport/internal/registry"
)

// Options controls port allocation behaviour.
type Options struct {
	// RangeMin and RangeMax override the framework range when both are non-zero.
	RangeMin int
	RangeMax int
}

// Allocate returns the port assigned to key. If an entry already exists for
// key that port is returned immediately (fixed-port policy). Otherwise a new
// port is probed and returned. The registry is NOT written by this function —
// the caller is responsible for persisting the result.
func Allocate(key, framework string, reg *registry.Registry, opts Options) (int, error) {
	// Fixed-port policy: return existing entry without probing.
	if entry, ok := reg.Entries[key]; ok {
		return entry.Port, nil
	}

	r := portRange(framework, reg, opts)
	used := usedPorts(reg)

	for port := r.Min; port <= r.Max; port++ {
		if used[port] {
			continue
		}
		if ProbePort(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", r.Min, r.Max)
}

// portRange returns the PortRange to use for allocation.
func portRange(framework string, reg *registry.Registry, opts Options) registry.PortRange {
	if opts.RangeMin > 0 && opts.RangeMax > 0 {
		return registry.PortRange{Min: opts.RangeMin, Max: opts.RangeMax}
	}
	if r, ok := reg.RangePolicy[framework]; ok {
		return r
	}
	return reg.RangePolicy["default"]
}

// usedPorts builds a set of all ports currently registered or reserved.
func usedPorts(reg *registry.Registry) map[int]bool {
	used := make(map[int]bool, len(reg.Entries)+len(reg.Reserved))
	for _, entry := range reg.Entries {
		used[entry.Port] = true
	}
	for _, p := range reg.Reserved {
		used[p] = true
	}
	return used
}
