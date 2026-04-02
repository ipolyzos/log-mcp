package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServer_LogTools_listedWithDefaults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	srv := NewWithConfig(Config{})
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	for _, want := range []string{tools.ToolNameReadLogs, tools.ToolNameSearchLogs, tools.ToolNameFilterLogs} {
		if !slices.Contains(names, want) {
			t.Fatalf("missing tool %q in %v", want, names)
		}
	}
}

func TestServer_LogTools_unconfiguredSource(t *testing.T) {
	t.Setenv("LOG_MCP_ROOT", "")
	ctx := context.Background()
	srv := New()
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tools.ToolNameReadLogs,
		Arguments: map[string]any{"stream_id": "/tmp/x.log"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error, got %+v", res)
	}
}

func TestServer_LogTools_readSearchFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	content := "" +
		`{"level":"INFO","msg":"boot","@timestamp":"2026-01-10T08:00:00Z"}` + "\n" +
		`{"level":"WARN","msg":"checkout slow","@timestamp":"2026-01-12T10:30:00Z"}` + "\n" +
		`{"level":"ERROR","msg":"payment fail","@timestamp":"2026-01-15T14:00:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := source.NewFileLogSource("t", root)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewWithConfig(Config{LogSource: fs})
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	readRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameReadLogs,
		Arguments: map[string]any{
			"stream_id": path,
			"max_lines": float64(2),
		},
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
	if len(readOut.Lines) != 2 || readOut.Lines[0].Level != "INFO" {
		t.Fatalf("read_logs lines: %+v", readOut.Lines)
	}

	searchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameSearchLogs,
		Arguments: map[string]any{
			"stream_id": path,
			"keyword":   "checkout",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if searchRes.IsError {
		t.Fatalf("search_logs: %+v", searchRes)
	}
	var searchOut tools.LogsToolOutput
	if err := decodeStructured(searchRes, &searchOut); err != nil {
		t.Fatal(err)
	}
	if len(searchOut.Lines) != 1 || searchOut.Lines[0].Level != "WARN" {
		t.Fatalf("search_logs: %+v", searchOut.Lines)
	}

	filterLevel, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameFilterLogs,
		Arguments: map[string]any{
			"stream_id": path,
			"level":     "ERROR",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filterLevel.IsError {
		t.Fatalf("filter_logs level: %+v", filterLevel)
	}
	var fl tools.LogsToolOutput
	if err := decodeStructured(filterLevel, &fl); err != nil {
		t.Fatal(err)
	}
	if len(fl.Lines) != 1 || !strings.Contains(strings.ToLower(fl.Lines[0].Message), "payment") {
		t.Fatalf("filter level: %+v", fl.Lines)
	}

	filterTime, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameFilterLogs,
		Arguments: map[string]any{
			"stream_id": path,
			"since":     "2026-01-11T00:00:00Z",
			"until":     "2026-01-14T23:59:59Z",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filterTime.IsError {
		t.Fatalf("filter_logs time: %+v", filterTime)
	}
	var ft tools.LogsToolOutput
	if err := decodeStructured(filterTime, &ft); err != nil {
		t.Fatal(err)
	}
	if len(ft.Lines) != 1 || ft.Lines[0].Level != "WARN" {
		t.Fatalf("filter time: %+v", ft.Lines)
	}
}

func decodeStructured(res *mcp.CallToolResult, dst any) error {
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
