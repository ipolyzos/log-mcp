package server

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/buildinfo"
	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config configures optional server dependencies.
type Config struct {
	// LogSource backs read_logs / search_logs / filter_logs. If nil, those
	// tools still register but return an error until configured.
	LogSource source.LogSource
}

// New returns an MCP server configured with the default tool registry and
// LOG_MCP_ROOT (if set) as the file log source.
func New() *mcp.Server {
	return NewWithConfig(Config{LogSource: logSourceFromEnv()})
}

// NewWithConfig returns an MCP server with an explicit [Config] (tests, custom wiring).
func NewWithConfig(cfg Config) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    buildinfo.Name,
		Version: buildinfo.Version,
	}, nil)
	reg := tools.NewRegistry()
	reg.Register(tools.RegisterHealthCheck)
	reg.Register(tools.RegisterVersionInfo)
	reg.Register(tools.RegisterLogTools(cfg.LogSource))
	reg.Apply(srv)
	return srv
}

func logSourceFromEnv() source.LogSource {
	raw := strings.TrimSpace(os.Getenv("LOG_MCP_ROOT"))
	if raw == "" {
		return nil
	}
	roots := filepath.SplitList(raw)
	var cleaned []string
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r != "" {
			cleaned = append(cleaned, r)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	fs, err := source.NewFileLogSource("filesystem", cleaned...)
	if err != nil {
		log.Printf("log-mcp: ignoring LOG_MCP_ROOT: %v", err)
		return nil
	}
	return fs
}

// Run serves MCP over stdin/stdout until the client disconnects or ctx is done.
func Run(ctx context.Context) error {
	return New().Run(ctx, &mcp.StdioTransport{})
}
