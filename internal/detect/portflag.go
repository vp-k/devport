package detect

// frameworkPortFlags maps frameworks to the CLI flag commonly used to
// override the dev server port for local development.
var frameworkPortFlags = map[string]string{
	"angular": "--port",
	"next":    "-p",
	"nuxt":    "--port",
	"remix":   "--port",
	"svelte":  "--port",
	"vite":    "--port",
}

// PortFlagFor returns the CLI port flag for the framework, or "" when exec
// should rely on env injection only.
func PortFlagFor(framework string) string {
	return frameworkPortFlags[framework]
}
