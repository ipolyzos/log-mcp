package server

import (
	"context"

	"github.com/ipolyzos/log-mcp/internal/buildinfo"
	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New returns an MCP server configured with the default tool registry.
func New() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    buildinfo.Name,
		Version: buildinfo.Version,
	}, nil)
	reg := tools.NewRegistry()
	reg.Register(tools.RegisterHealthCheck)
	reg.Register(tools.RegisterVersionInfo)
	reg.Apply(srv)
	return srv
}

// Run serves MCP over stdin/stdout until the client disconnects or ctx is done.
func Run(ctx context.Context) error {
	return New().Run(ctx, &mcp.StdioTransport{})
}
