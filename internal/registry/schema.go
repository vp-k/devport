package registry

import "time"

// KeySource indicates how the project key was resolved.
type KeySource string

const (
	KeySourcePackageJSON KeySource = "package.json"
	KeySourceGitRemote   KeySource = "git-remote"
	KeySourcePath        KeySource = "path"
)

// PortRange defines min/max port boundaries.
type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Meta stores registry-level metadata.
type Meta struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Entry represents a single project-to-port mapping.
type Entry struct {
	Port           int       `json:"port"`
	KeySource      KeySource `json:"keySource"`
	DisplayName    string    `json:"displayName"`
	ProjectPath    string    `json:"projectPath"`
	Framework      string    `json:"framework,omitempty"`
	AllocatedAt    time.Time `json:"allocatedAt"`
	LastAccessedAt time.Time `json:"lastAccessedAt"`
	RangeMin       int       `json:"rangeMin"`
	RangeMax       int       `json:"rangeMax"`
}

// Registry is the root structure stored in ~/.devports.json.
type Registry struct {
	Version     int                  `json:"version"`
	Meta        Meta                 `json:"meta"`
	Entries     map[string]*Entry    `json:"entries"`
	Reserved    []int                `json:"reserved"`
	RangePolicy map[string]PortRange `json:"rangePolicy"`
}

// DefaultRangePolicy returns the built-in framework port ranges.
func DefaultRangePolicy() map[string]PortRange {
	return map[string]PortRange{
		"default": {Min: 3000, Max: 9999},
		"next":    {Min: 3000, Max: 3999},
		"vite":    {Min: 5000, Max: 5999},
		"express": {Min: 4000, Max: 4999},
		"angular": {Min: 4200, Max: 4299},
		"nest":    {Min: 3000, Max: 3999},
		"cra":     {Min: 3000, Max: 3999},
		"go":      {Min: 8000, Max: 8999},
		"gin":     {Min: 8000, Max: 8999},
		"echo":    {Min: 8000, Max: 8999},
		"fiber":   {Min: 8000, Max: 8999},
		"chi":     {Min: 8000, Max: 8999},
		"bun":     {Min: 3000, Max: 3999},
		"deno":    {Min: 3000, Max: 3999},
	}
}
