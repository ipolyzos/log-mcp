package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameHealthCheck is the MCP tools/call name for the health probe.
const ToolNameHealthCheck = "health_check"

type healthInput struct{}

// HealthOutput is the structured result for health_check.
type HealthOutput struct {
	Status string `json:"status" jsonschema:"health status string, e.g. ok"`
}

// RegisterHealthCheck registers the health_check tool on s.
func RegisterHealthCheck(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        ToolNameHealthCheck,
		Description: "Returns whether the log-mcp server is running and able to handle tool calls.",
	}, healthCheck)
}

func healthCheck(_ context.Context, _ *mcp.CallToolRequest, _ healthInput) (*mcp.CallToolResult, HealthOutput, error) {
	return nil, HealthOutput{Status: "ok"}, nil
}
