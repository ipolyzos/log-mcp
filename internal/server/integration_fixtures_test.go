package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestIntegration_stdio_allParseFixtures_logTools exercises read_logs,
// summarize_errors, and detect_anomalies over every *.log under
// internal/parse/testdata/fixtures via the same stdio-style MCP path as hosts.
func TestIntegration_stdio_allParseFixtures_logTools(t *testing.T) {
	root := repoRootFromServerTest(t)
	fixDir := filepath.Join(root, "internal", "parse", "testdata", "fixtures")
	ents, err := os.ReadDir(fixDir)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".log") {
			continue
		}
		paths = append(paths, filepath.Join(fixDir, e.Name()))
	}
	sort.Strings(paths)
	if len(paths) < 10 {
		t.Fatalf("fixtures: want many .log files, got %d under %s", len(paths), fixDir)
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

	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-fixtures", Version: "0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	for _, logPath := range paths {
		base := filepath.Base(logPath)
		t.Run(base, func(t *testing.T) {
			readJSON := fmt.Sprintf(`{"stream_id":%q,"max_lines":2000}`, logPath)
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
			if len(readOut.Lines) < 1 {
				t.Fatalf("read_logs: want >=1 line, got 0 (stream may be unreadable or empty)")
			}

			sumJSON := fmt.Sprintf(`{"stream_id":%q}`, logPath)
			var sumArgs map[string]any
			if err := json.Unmarshal([]byte(sumJSON), &sumArgs); err != nil {
				t.Fatal(err)
			}
			sumRes, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tools.ToolNameSummarizeErrors,
				Arguments: sumArgs,
			})
			if err != nil {
				t.Fatal(err)
			}
			if sumRes.IsError {
				t.Fatalf("summarize_errors: %+v", sumRes)
			}
			var sumOut tools.SummarizeErrorsOutput
			if err := decodeStructured(sumRes, &sumOut); err != nil {
				t.Fatal(err)
			}
			if sumOut.ScannedLines < 1 {
				t.Fatalf("summarize_errors: want scanned_lines >= 1, got %d", sumOut.ScannedLines)
			}

			detJSON := fmt.Sprintf(`{"stream_id":%q}`, logPath)
			var detArgs map[string]any
			if err := json.Unmarshal([]byte(detJSON), &detArgs); err != nil {
				t.Fatal(err)
			}
			detRes, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tools.ToolNameDetectAnomalies,
				Arguments: detArgs,
			})
			if err != nil {
				t.Fatal(err)
			}
			if detRes.IsError {
				t.Fatalf("detect_anomalies: %+v", detRes)
			}
			var detOut tools.DetectAnomaliesOutput
			if err := decodeStructured(detRes, &detOut); err != nil {
				t.Fatal(err)
			}
			if detOut.ScannedLines < 1 {
				t.Fatalf("detect_anomalies: want scanned_lines >= 1, got %d", detOut.ScannedLines)
			}

			// search_logs: common access-log token (most fixtures include HTTP).
			// extract_trace_flow on OTel hex trace_id is not asserted here: pipeline Parsed
			// is re-derived from redacted text, so long ids may not match correlation.
			srchJSON := fmt.Sprintf(`{"stream_id":%q,"keyword":"HTTP","max_matches":5}`, logPath)
			var srchArgs map[string]any
			if err := json.Unmarshal([]byte(srchJSON), &srchArgs); err != nil {
				t.Fatal(err)
			}
			srchRes, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tools.ToolNameSearchLogs,
				Arguments: srchArgs,
			})
			if err != nil {
				t.Fatal(err)
			}
			if srchRes.IsError {
				t.Fatalf("search_logs: %+v", srchRes)
			}
			var srchOut tools.LogsToolOutput
			if err := decodeStructured(srchRes, &srchOut); err != nil {
				t.Fatal(err)
			}
			if len(srchOut.Lines) < 1 {
				t.Fatalf("search_logs: want >=1 hit for keyword HTTP, got 0")
			}
		})
	}

	cancel()
	<-errCh
}
