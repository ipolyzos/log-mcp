package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/buildinfo"
	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stdioPairedTransports returns server and client [mcp.IOTransport] pairs wired
// through pipes. The framing is newline-delimited JSON — the same as
// [mcp.StdioTransport], so this exercises the real JSON-RPC path hosts use.
func stdioPairedTransports() (srvT, cliT *mcp.IOTransport) {
	cr, cw := io.Pipe()
	sr, sw := io.Pipe()
	// Server reads what the client writes; client reads what the server writes.
	srvT = &mcp.IOTransport{Reader: cr, Writer: sw}
	cliT = &mcp.IOTransport{Reader: sr, Writer: cw}
	return srvT, cliT
}

// runMCPServerStdioLike runs srv over transport until ctx is cancelled or the
// client disconnects. Errors are sent to errCh (at most one).
func runMCPServerStdioLike(ctx context.Context, srv *mcp.Server, srvT *mcp.IOTransport, errCh chan<- error) {
	go func() {
		err := srv.Run(ctx, srvT)
		select {
		case errCh <- err:
		default:
		}
	}()
}

func TestIntegration_stdioToolsCall_filterLogsFromJSONArgs(t *testing.T) {
	// t.Setenv is incompatible with parallel tests that share keys.
	root := t.TempDir()
	logPath := filepath.Join(root, "app.log")
	content := "" +
		`{"level":"INFO","msg":"boot","@timestamp":"2026-01-10T08:00:00Z"}` + "\n" +
		`{"level":"WARN","msg":"checkout slow","@timestamp":"2026-01-12T10:30:00Z"}` + "\n" +
		`{"level":"ERROR","msg":"payment fail","@timestamp":"2026-01-15T14:00:00Z"}` + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LOG_MCP_ROOT", root)
	t.Setenv("LOG_MCP_K8S_NAMESPACES", "")
	t.Setenv("LOG_MCP_DOCKER_CONTAINERS", "")
	t.Setenv("LOG_MCP_CF_APPS", "")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srvT, cliT := stdioPairedTransports()
	srv := New()
	errCh := make(chan error, 1)
	runMCPServerStdioLike(ctx, srv, srvT, errCh)

	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-itest", Version: "0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	// Full JSON object as a host would send (quoted strings, integer max_lines).
	filterJSON := fmt.Sprintf(
		`{"stream_id":%q,"level":"ERROR","keyword":"payment","since":"2026-01-01T00:00:00Z","until":"2026-01-20T23:59:59Z","max_lines":100}`,
		logPath,
	)
	var args map[string]any
	if err := json.Unmarshal([]byte(filterJSON), &args); err != nil {
		t.Fatalf("filter_logs JSON: %v", err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tools.ToolNameFilterLogs,
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("filter_logs: %+v", res)
	}
	var out tools.LogsToolOutput
	if err := decodeStructured(res, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Lines) != 1 {
		t.Fatalf("want 1 line, got %d: %+v", len(out.Lines), out.Lines)
	}
	if out.ResultHint != "" {
		t.Fatalf("unexpected result_hint: %q", out.ResultHint)
	}
	if out.Lines[0].Level != "ERROR" || !strings.Contains(out.Lines[0].Message, "payment") {
		t.Fatalf("unexpected line: %+v", out.Lines[0])
	}

	cancel()
	<-errCh
}

func TestIntegration_stdioToolsCall_readLogsAndMeta(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "lines.log")
	if err := os.WriteFile(logPath, []byte("plain info line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOG_MCP_ROOT", root)
	t.Setenv("LOG_MCP_K8S_NAMESPACES", "")
	t.Setenv("LOG_MCP_DOCKER_CONTAINERS", "")
	t.Setenv("LOG_MCP_CF_APPS", "")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srvT, cliT := stdioPairedTransports()
	srv := New()
	errCh := make(chan error, 1)
	runMCPServerStdioLike(ctx, srv, srvT, errCh)

	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-itest", Version: "0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	readJSON := fmt.Sprintf(`{"stream_id":%q,"max_lines":10}`, logPath)
	var readArgs map[string]any
	if err := json.Unmarshal([]byte(readJSON), &readArgs); err != nil {
		t.Fatal(err)
	}
	readRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tools.ToolNameReadLogs,
		Arguments: readArgs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if readRes.IsError {
		t.Fatalf("read_logs: %+v", readRes)
	}
	var readOut tools.LogsToolOutput
	if err := decodeStructured(readRes, &readOut); err != nil {
		t.Fatal(err)
	}
	if len(readOut.Lines) != 1 {
		t.Fatalf("read_logs lines: %+v", readOut.Lines)
	}
	if readOut.ResultHint != "" {
		t.Fatalf("unexpected result_hint: %q", readOut.ResultHint)
	}

	hc, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tools.ToolNameHealthCheck})
	if err != nil {
		t.Fatal(err)
	}
	if hc.IsError {
		t.Fatalf("health_check: %+v", hc)
	}

	vi, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tools.ToolNameVersionInfo})
	if err != nil {
		t.Fatal(err)
	}
	if vi.IsError {
		t.Fatalf("version_info: %+v", vi)
	}
	var ver tools.VersionOutput
	if err := decodeStructured(vi, &ver); err != nil {
		t.Fatal(err)
	}
	if ver.Name != buildinfo.Name || ver.Version != buildinfo.Version {
		t.Fatalf("version_info: got %+v want name=%q version=%q", ver, buildinfo.Name, buildinfo.Version)
	}

	cancel()
	<-errCh
}

func TestIntegration_stdio_readLogs_emptyStreamResultHint(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "empty.log")
	if err := os.WriteFile(logPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOG_MCP_ROOT", root)
	t.Setenv("LOG_MCP_K8S_NAMESPACES", "")
	t.Setenv("LOG_MCP_DOCKER_CONTAINERS", "")
	t.Setenv("LOG_MCP_CF_APPS", "")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srvT, cliT := stdioPairedTransports()
	srv := New()
	errCh := make(chan error, 1)
	runMCPServerStdioLike(ctx, srv, srvT, errCh)

	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-itest", Version: "0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	readJSON := fmt.Sprintf(`{"stream_id":%q,"max_lines":10}`, logPath)
	var readArgs map[string]any
	if err := json.Unmarshal([]byte(readJSON), &readArgs); err != nil {
		t.Fatal(err)
	}
	readRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tools.ToolNameReadLogs,
		Arguments: readArgs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if readRes.IsError {
		t.Fatalf("read_logs: %+v", readRes)
	}
	var readOut tools.LogsToolOutput
	if err := decodeStructured(readRes, &readOut); err != nil {
		t.Fatal(err)
	}
	if len(readOut.Lines) != 0 {
		t.Fatalf("expected no lines, got %d", len(readOut.Lines))
	}
	if readOut.ResultHint != "no_lines_in_stream" {
		t.Fatalf("result_hint: got %q", readOut.ResultHint)
	}

	cancel()
	<-errCh
}

// repoRootFromServerTest returns the repository root when tests run with cwd
// set to internal/server (normal `go test ./internal/server/...`).
func repoRootFromServerTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// TestIntegration_stdioToolsCall_filterLogsKubernetesCRI_fixture runs the same
// filter_logs JSON shapes recommended for MCP clients (time window, then AND
// predicates) against the checked-in Kubernetes CRI fixture.
func TestIntegration_stdioToolsCall_filterLogsKubernetesCRI_fixture(t *testing.T) {
	root := repoRootFromServerTest(t)
	logPath := filepath.Join(root, "internal", "parse", "testdata", "fixtures", "01_kubernetes_cri.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	t.Setenv("LOG_MCP_ROOT", root)
	t.Setenv("LOG_MCP_K8S_NAMESPACES", "")
	t.Setenv("LOG_MCP_DOCKER_CONTAINERS", "")
	t.Setenv("LOG_MCP_CF_APPS", "")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srvT, cliT := stdioPairedTransports()
	srv := New()
	errCh := make(chan error, 1)
	runMCPServerStdioLike(ctx, srv, srvT, errCh)

	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-itest", Version: "0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	t.Run("time_window", func(t *testing.T) {
		filterJSON := fmt.Sprintf(
			`{"stream_id":%q,"since":"2024-06-15T10:30:00Z","until":"2024-06-15T10:30:10Z","max_lines":20}`,
			logPath,
		)
		var args map[string]any
		if err := json.Unmarshal([]byte(filterJSON), &args); err != nil {
			t.Fatal(err)
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      tools.ToolNameFilterLogs,
			Arguments: args,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IsError {
			t.Fatalf("filter_logs: %+v", res)
		}
		var out tools.LogsToolOutput
		if err := decodeStructured(res, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.Lines) < 1 {
			t.Fatalf("want at least 1 line in window, got %d: %+v", len(out.Lines), out.Lines)
		}
	})

	t.Run("level_keyword_and_time", func(t *testing.T) {
		filterJSON := fmt.Sprintf(
			`{"stream_id":%q,"since":"2024-06-01T00:00:00Z","until":"2025-01-01T00:00:00Z","level":"ERROR","keyword":"upstream","max_lines":50}`,
			logPath,
		)
		var args map[string]any
		if err := json.Unmarshal([]byte(filterJSON), &args); err != nil {
			t.Fatal(err)
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      tools.ToolNameFilterLogs,
			Arguments: args,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IsError {
			t.Fatalf("filter_logs: %+v", res)
		}
		var out tools.LogsToolOutput
		if err := decodeStructured(res, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.Lines) != 1 {
			t.Fatalf("want 1 ERROR+upstream line (503), got %d: %+v", len(out.Lines), out.Lines)
		}
		if out.Lines[0].Level != "ERROR" {
			t.Fatalf("want ERROR level, got %q line=%+v", out.Lines[0].Level, out.Lines[0])
		}
	})

	cancel()
	<-errCh
}
