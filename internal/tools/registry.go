package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Installer registers MCP tools on a server instance.
type Installer func(*mcp.Server)

// Registry collects installers so the server can apply them in one place.
type Registry struct {
	installers []Installer
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register appends an installer (typically one per tool family).
func (r *Registry) Register(fn Installer) {
	if fn == nil {
		return
	}
	r.installers = append(r.installers, fn)
}

// Apply runs all registered installers against s.
func (r *Registry) Apply(s *mcp.Server) {
	for _, fn := range r.installers {
		fn(s)
	}
}

// InstallerCount returns how many installers are registered (for tests).
func (r *Registry) InstallerCount() int {
	return len(r.installers)
}
