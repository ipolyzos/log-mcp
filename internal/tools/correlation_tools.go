package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/logcorr"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameGroupByField is the MCP tools/call name for grouping by a log field.
const ToolNameGroupByField = "group_by_field"

// ToolNameCorrelateEvents is the MCP tools/call name for correlating by field value.
const ToolNameCorrelateEvents = "correlate_events"

type groupByFieldInput struct {
	StreamID     string `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	Field        string `json:"field" jsonschema:"required logical field: trace_id, request_id, or a JSON key name"`
	MaxScanLines *int64 `json:"max_scan_lines,omitempty" jsonschema:"max lines to scan; omit for default"`
	MaxGroups    *int64 `json:"max_groups,omitempty" jsonschema:"max distinct groups to return; omit for default"`
}

// GroupByFieldOutput is the structured result for group_by_field.
type GroupByFieldOutput struct {
	StreamID     string              `json:"stream_id"`
	Field        string              `json:"field"`
	ScannedLines int64               `json:"scanned_lines"`
	Truncated    bool                `json:"truncated"`
	Groups       []GroupByFieldGroup `json:"groups"`
}

// GroupByFieldGroup is one bucket of lines sharing the same field value.
type GroupByFieldGroup struct {
	Value       string  `json:"value"`
	Count       int64   `json:"count"`
	FirstLine   int64   `json:"first_line"`
	LastLine    int64   `json:"last_line"`
	LineNumbers []int64 `json:"line_numbers,omitempty"`
}

type correlateEventsInput struct {
	StreamID     string `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	Field        string `json:"field" jsonschema:"required logical field: trace_id, request_id, or a JSON key name"`
	Value        string `json:"value" jsonschema:"required field value to correlate (e.g. one trace or request id)"`
	MaxScanLines *int64 `json:"max_scan_lines,omitempty" jsonschema:"max lines to scan; omit for default"`
	MaxEvents    *int64 `json:"max_events,omitempty" jsonschema:"max events returned; omit for default"`
}

// CorrelateEventsOutput is the structured result for correlate_events.
type CorrelateEventsOutput struct {
	StreamID     string        `json:"stream_id"`
	Field        string        `json:"field"`
	Value        string        `json:"value"`
	ScannedLines int64         `json:"scanned_lines"`
	Truncated    bool          `json:"truncated"`
	Events       []LogLineJSON `json:"events"`
}

func registerCorrelationTools(s *mcp.Server, src source.LogSource) {
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameGroupByField,
		Description: "Scan redacted logs and group lines by a structured field. Supports logical names trace_id and request_id " +
			"(plus common aliases like traceId, x-request-id). Lines missing the field are skipped. " +
			"Returns groups sorted by count descending. Caps scan (default 20000) and returned groups (default 50).",
	}, groupByFieldHandler(src))
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameCorrelateEvents,
		Description: "Return all log lines in scan order sorted by timestamp (then line number) that share the same field value " +
			"(trace_id, request_id, or custom JSON key). Use after group_by_field to drill into one id. " +
			"Caps scan and max events (default 500).",
	}, correlateEventsHandler(src))
}

func groupByFieldHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, groupByFieldInput) (*mcp.CallToolResult, GroupByFieldOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in groupByFieldInput) (*mcp.CallToolResult, GroupByFieldOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, GroupByFieldOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, GroupByFieldOutput{}, fmt.Errorf("stream_id is required")
		}
		field := strings.TrimSpace(in.Field)
		if field == "" {
			return nil, GroupByFieldOutput{}, fmt.Errorf("field is required")
		}
		res, err := logcorr.GroupByField(ctx, src, sid, field, ptrInt64(in.MaxScanLines), ptrInt64(in.MaxGroups))
		if err != nil {
			return nil, GroupByFieldOutput{}, err
		}
		groups := make([]GroupByFieldGroup, 0, len(res.Groups))
		for _, g := range res.Groups {
			groups = append(groups, GroupByFieldGroup{
				Value:       g.Value,
				Count:       g.Count,
				FirstLine:   g.FirstLine,
				LastLine:    g.LastLine,
				LineNumbers: g.LineNumbers,
			})
		}
		return nil, GroupByFieldOutput{
			StreamID:     res.StreamID,
			Field:        res.Field,
			ScannedLines: res.ScannedLines,
			Truncated:    res.Truncated,
			Groups:       groups,
		}, nil
	}
}

func correlateEventsHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, correlateEventsInput) (*mcp.CallToolResult, CorrelateEventsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in correlateEventsInput) (*mcp.CallToolResult, CorrelateEventsOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, CorrelateEventsOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, CorrelateEventsOutput{}, fmt.Errorf("stream_id is required")
		}
		field := strings.TrimSpace(in.Field)
		if field == "" {
			return nil, CorrelateEventsOutput{}, fmt.Errorf("field is required")
		}
		res, err := logcorr.CorrelateEvents(ctx, src, sid, field, in.Value, ptrInt64(in.MaxScanLines), ptrInt64(in.MaxEvents))
		if err != nil {
			return nil, CorrelateEventsOutput{}, err
		}
		events := make([]LogLineJSON, 0, len(res.Events))
		for _, e := range res.Events {
			events = append(events, resultToLogLineJSON(e.Result))
		}
		return nil, CorrelateEventsOutput{
			StreamID:     res.StreamID,
			Field:        res.Field,
			Value:        res.Value,
			ScannedLines: res.ScannedLines,
			Truncated:    res.Truncated,
			Events:       events,
		}, nil
	}
}

func resultToLogLineJSON(r pipeline.Result) LogLineJSON {
	j := LogLineJSON{
		LineNumber: r.LineNumber,
		Level:      r.Parsed.Level,
		Message:    r.Parsed.Message,
		Text:       r.Redacted,
	}
	if r.Parsed.Timestamp != nil {
		j.Timestamp = r.Parsed.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return j
}
