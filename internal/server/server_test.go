package server

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/buildinfo"
	"github.com/ipolyzos/log-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServer_DefaultToolsList(t *testing.T) {
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

	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	want := []string{
		tools.ToolNameCorrelateEvents,
		tools.ToolNameDetectAnomalies,
		tools.ToolNameExplainLogPattern,
		tools.ToolNameExtractTraceFlow,
		tools.ToolNameFilterLogs,
		tools.ToolNameGroupByField,
		tools.ToolNameHealthCheck,
		tools.ToolNameIncidentSummary,
		tools.ToolNameReadLogs,
		tools.ToolNameSearchLogs,
		tools.ToolNameSummarizeErrors,
		tools.ToolNameVersionInfo,
	}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Fatalf("tool names: got %v want %v", names, want)
	}
}

func TestServer_LLMReasoningTools_noLogSource(t *testing.T) {
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

	isRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameIncidentSummary,
		Arguments: map[string]any{
			"structured_summary": map[string]any{
				"groups": []any{
					map[string]any{"rank": 1, "count": 2, "pattern": "timeout"},
				},
				"error_line_count": float64(2),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if isRes.IsError {
		t.Fatalf("incident_summary: %+v", isRes)
	}
	b, err := json.Marshal(isRes.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var isOut tools.IncidentSummaryOutput
	if err := json.Unmarshal(b, &isOut); err != nil {
		t.Fatal(err)
	}
	if isOut.Headline == "" || len(isOut.Bullets) == 0 {
		t.Fatalf("incident_summary output: %+v", isOut)
	}

	epRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: tools.ToolNameExplainLogPattern,
		Arguments: map[string]any{
			"error_group": map[string]any{
				"rank": float64(1), "count": float64(5), "pattern": "boom", "fingerprint": "fp1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if epRes.IsError {
		t.Fatalf("explain_log_pattern: %+v", epRes)
	}
	b2, err := json.Marshal(epRes.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var epOut tools.ExplainLogPatternOutput
	if err := json.Unmarshal(b2, &epOut); err != nil {
		t.Fatal(err)
	}
	if epOut.Overview == "" {
		t.Fatalf("explain output: %+v", epOut)
	}
}

func TestServer_HealthCheckTool(t *testing.T) {
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
		Name:      tools.ToolNameHealthCheck,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out tools.HealthOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "ok" {
		t.Fatalf("status: got %q", out.Status)
	}
}

func TestServer_VersionInfoTool(t *testing.T) {
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
		Name:      tools.ToolNameVersionInfo,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out tools.VersionOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != buildinfo.Name {
		t.Fatalf("name: got %q want %q", out.Name, buildinfo.Name)
	}
	if out.Version != buildinfo.Version {
		t.Fatalf("version: got %q want %q", out.Version, buildinfo.Version)
	}
}
