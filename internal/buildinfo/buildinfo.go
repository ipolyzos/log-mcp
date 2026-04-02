// Package buildinfo holds static distribution metadata for the log-mcp binary.
package buildinfo

const (
	// Name is the MCP server implementation name.
	Name = "log-mcp"
	// Version is the server semantic version (override with -ldflags for releases).
	Version = "0.1.0"
)
