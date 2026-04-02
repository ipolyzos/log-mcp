package tools

import (
	"context"

	"github.com/ipolyzos/log-mcp/internal/buildinfo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameVersionInfo is the MCP tools/call name for version metadata.
const ToolNameVersionInfo = "version_info"

type versionInput struct{}

// VersionOutput is the structured result for version_info.
type VersionOutput struct {
	Name    string `json:"name" jsonschema:"MCP server implementation name"`
	Version string `json:"version" jsonschema:"server version string"`
}

// RegisterVersionInfo registers the version_info tool on s.
func RegisterVersionInfo(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        ToolNameVersionInfo,
		Description: "Returns the log-mcp server name and version.",
	}, versionInfo)
}

func versionInfo(_ context.Context, _ *mcp.CallToolRequest, _ versionInput) (*mcp.CallToolResult, VersionOutput, error) {
	return nil, VersionOutput{
		Name:    buildinfo.Name,
		Version: buildinfo.Version,
	}, nil
}
