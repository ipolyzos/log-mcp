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
	// LogSource backs read/search/filter logs, summarize_errors, and correlation tools.
	// If nil, those tools still register but return an error until configured.
	LogSource source.LogSource
}

// New returns an MCP server configured with the default tool registry and
// log sources from the environment ([logSourceFromEnv]).
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
	fileSrc := fileSourceFromEnv()
	k8sSrc := k8sSourceFromEnv()
	switch {
	case fileSrc != nil && k8sSrc != nil:
		multi, err := source.NewMultiLogSource("combined", fileSrc, k8sSrc)
		if err != nil {
			log.Printf("log-mcp: combined log source: %v", err)
			return fileSrc
		}
		return multi
	case k8sSrc != nil:
		return k8sSrc
	default:
		return fileSrc
	}
}

func fileSourceFromEnv() *source.FileLogSource {
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

func k8sSourceFromEnv() *source.K8sLogSource {
	raw := strings.TrimSpace(os.Getenv("LOG_MCP_K8S_NAMESPACES"))
	if raw == "" {
		return nil
	}
	var ns []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ns = append(ns, part)
		}
	}
	if len(ns) == 0 {
		return nil
	}
	runner := &source.ExecKubectlRunner{
		Path: strings.TrimSpace(os.Getenv("LOG_MCP_K8S_KUBECTL")),
	}
	k, err := source.NewK8sLogSource("kubernetes", ns, runner)
	if err != nil {
		log.Printf("log-mcp: ignoring LOG_MCP_K8S_NAMESPACES: %v", err)
		return nil
	}
	return k
}

// Run serves MCP over stdin/stdout until the client disconnects or ctx is done.
func Run(ctx context.Context) error {
	return New().Run(ctx, &mcp.StdioTransport{})
}
