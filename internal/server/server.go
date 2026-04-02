package server

import (
	"context"
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
	// LogSource backs read/search/filter logs, summarize_errors, correlation, and analysis tools.
	// If nil, those tools still register but return an error until configured.
	// incident_summary and explain_log_pattern do not use LogSource (structured JSON only).
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
	reg.Register(tools.RegisterLLMTools)
	reg.Apply(srv)
	return srv
}

func logSourceFromEnv() source.LogSource {
	fileSrc := fileSourceFromEnv()
	k8sSrc := k8sSourceFromEnv()
	dockerSrc := dockerSourceFromEnv()
	cfSrc := cfSourceFromEnv()

	n := 0
	if fileSrc != nil {
		n++
	}
	if k8sSrc != nil {
		n++
	}
	if dockerSrc != nil {
		n++
	}
	if cfSrc != nil {
		n++
	}
	if n == 0 {
		return nil
	}
	if n == 1 {
		switch {
		case fileSrc != nil:
			return fileSrc
		case k8sSrc != nil:
			return k8sSrc
		case dockerSrc != nil:
			return dockerSrc
		default:
			return cfSrc
		}
	}
	multi, err := source.NewMultiLogSource("combined", fileSrc, k8sSrc, dockerSrc, cfSrc)
	if err != nil {
		logger().Warn("log-mcp: combined log source", "error", err)
		return firstNonNilLogSource(fileSrc, k8sSrc, dockerSrc, cfSrc)
	}
	return multi
}

func firstNonNilLogSource(fileSrc *source.FileLogSource, k8sSrc *source.K8sLogSource, dockerSrc *source.DockerLogSource, cfSrc *source.CFLogSource) source.LogSource {
	if fileSrc != nil {
		return fileSrc
	}
	if k8sSrc != nil {
		return k8sSrc
	}
	if dockerSrc != nil {
		return dockerSrc
	}
	return cfSrc
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
		logger().Warn("log-mcp: ignoring LOG_MCP_ROOT", "error", err)
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
		logger().Warn("log-mcp: ignoring LOG_MCP_K8S_NAMESPACES", "error", err)
		return nil
	}
	return k
}

func dockerSourceFromEnv() *source.DockerLogSource {
	raw := strings.TrimSpace(os.Getenv("LOG_MCP_DOCKER_CONTAINERS"))
	if raw == "" {
		return nil
	}
	var names []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	if len(names) == 0 {
		return nil
	}
	runner := &source.ExecDockerRunner{
		Path: strings.TrimSpace(os.Getenv("LOG_MCP_DOCKER_DOCKER")),
	}
	d, err := source.NewDockerLogSource("docker", names, runner)
	if err != nil {
		logger().Warn("log-mcp: ignoring LOG_MCP_DOCKER_CONTAINERS", "error", err)
		return nil
	}
	return d
}

func cfSourceFromEnv() *source.CFLogSource {
	raw := strings.TrimSpace(os.Getenv("LOG_MCP_CF_APPS"))
	if raw == "" {
		return nil
	}
	var apps []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			apps = append(apps, part)
		}
	}
	if len(apps) == 0 {
		return nil
	}
	runner := &source.ExecCFRunner{
		Path: strings.TrimSpace(os.Getenv("LOG_MCP_CF_CLI")),
	}
	c, err := source.NewCFLogSource("cloudfoundry", apps, runner)
	if err != nil {
		logger().Warn("log-mcp: ignoring LOG_MCP_CF_APPS", "error", err)
		return nil
	}
	return c
}

// Run serves MCP over stdin/stdout until the client disconnects or ctx is done.
func Run(ctx context.Context) error {
	return New().Run(ctx, &mcp.StdioTransport{})
}
