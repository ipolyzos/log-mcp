package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/logtool"
	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool names for log retrieval MCP tools.
const (
	ToolNameReadLogs   = "read_logs"
	ToolNameSearchLogs = "search_logs"
	ToolNameFilterLogs = "filter_logs"
)

const (
	defaultReadMax    int64 = 500
	defaultSearchMax  int64 = 500
	defaultFilterMax  int64 = 500
	maxOutputLinesCap int64 = 10_000
)

// readLogsInput is the payload for read_logs.
type readLogsInput struct {
	StreamID string  `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	MaxLines *int64  `json:"max_lines,omitempty" jsonschema:"max lines to return; omit for default"`
	Since    *string `json:"since,omitempty" jsonschema:"optional RFC3339 lower time bound (inclusive)"`
	Until    *string `json:"until,omitempty" jsonschema:"optional RFC3339 upper time bound (inclusive)"`
	Level    *string `json:"level,omitempty" jsonschema:"optional log level e.g. INFO WARN ERROR"`
	Keyword  *string `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring on redacted line"`
}

// searchLogsInput is the payload for search_logs.
type searchLogsInput struct {
	StreamID   string  `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	Keyword    string  `json:"keyword" jsonschema:"required case-insensitive substring to find in redacted lines"`
	MaxMatches *int64  `json:"max_matches,omitempty" jsonschema:"max matching lines; omit for default"`
	Since      *string `json:"since,omitempty" jsonschema:"optional RFC3339 lower time bound (inclusive)"`
	Until      *string `json:"until,omitempty" jsonschema:"optional RFC3339 upper time bound (inclusive)"`
	Level      *string `json:"level,omitempty" jsonschema:"optional log level e.g. INFO WARN ERROR"`
}

// filterLogsInput is the payload for filter_logs.
type filterLogsInput struct {
	StreamID string  `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	MaxLines *int64  `json:"max_lines,omitempty" jsonschema:"max matching lines to return; omit for default"`
	Since    *string `json:"since,omitempty" jsonschema:"optional RFC3339 lower time bound (inclusive)"`
	Until    *string `json:"until,omitempty" jsonschema:"optional RFC3339 upper time bound (inclusive)"`
	Level    *string `json:"level,omitempty" jsonschema:"optional log level e.g. INFO WARN ERROR"`
	Keyword  *string `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring on redacted line"`
}

// LogsToolOutput is the structured result for read_logs, search_logs, and filter_logs.
type LogsToolOutput struct {
	StreamID  string        `json:"stream_id"`
	Lines     []LogLineJSON `json:"lines"`
	Truncated bool          `json:"truncated"`
}

// LogLineJSON is one log row in tool output (timestamps as RFC3339 strings).
type LogLineJSON struct {
	LineNumber int64  `json:"line_number"`
	Timestamp  string `json:"timestamp,omitempty"`
	Level      string `json:"level,omitempty"`
	Message    string `json:"message,omitempty"`
	Text       string `json:"text"`
}

// RegisterLogTools registers read_logs, search_logs, and filter_logs. If src is
// nil, handlers return a tool error asking to configure LOG_MCP_ROOT.
func RegisterLogTools(src source.LogSource) Installer {
	return func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        ToolNameReadLogs,
			Description: "Read log lines from a stream with optional limits and filters (time range, level, keyword). Lines are redacted. Use stream_id from the source (e.g. absolute file path under LOG_MCP_ROOT). When no filters are set, reads sequentially up to max_lines without scanning the whole file.",
		}, readLogsHandler(src))
		mcp.AddTool(s, &mcp.Tool{
			Name:        ToolNameSearchLogs,
			Description: "Search redacted log lines for a keyword (case-insensitive), with optional time range and level filters. Returns up to max_matches hits.",
		}, searchLogsHandler(src))
		mcp.AddTool(s, &mcp.Tool{
			Name:        ToolNameFilterLogs,
			Description: "Return log lines matching at least one filter: time range (since/until), level, and/or keyword. All conditions are ANDed. Lines are redacted.",
		}, filterLogsHandler(src))
	}
}

func readLogsHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, readLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in readLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, LogsToolOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, LogsToolOutput{}, fmt.Errorf("stream_id is required")
		}
		max := clampMax(ptrInt64(in.MaxLines), defaultReadMax)
		f, err := filterFromStrings(ptrStr(in.Since), ptrStr(in.Until), ptrStr(in.Level), ptrStr(in.Keyword))
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		q := source.Query{StreamID: sid}
		if f.Empty() {
			q.MaxLines = max
		}
		lines, trunc, err := logtool.Collect(ctx, src, q, f, max)
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		return nil, toLogsOutput(sid, lines, trunc), nil
	}
}

func searchLogsHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, searchLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in searchLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, LogsToolOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, LogsToolOutput{}, fmt.Errorf("stream_id is required")
		}
		kw := strings.TrimSpace(in.Keyword)
		if kw == "" {
			return nil, LogsToolOutput{}, fmt.Errorf("keyword is required for search_logs")
		}
		max := clampMax(ptrInt64(in.MaxMatches), defaultSearchMax)
		f, err := filterFromStrings(ptrStr(in.Since), ptrStr(in.Until), ptrStr(in.Level), kw)
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		q := source.Query{StreamID: sid}
		lines, trunc, err := logtool.Collect(ctx, src, q, f, max)
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		return nil, toLogsOutput(sid, lines, trunc), nil
	}
}

func filterLogsHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, filterLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in filterLogsInput) (*mcp.CallToolResult, LogsToolOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, LogsToolOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, LogsToolOutput{}, fmt.Errorf("stream_id is required")
		}
		f, err := filterFromStrings(ptrStr(in.Since), ptrStr(in.Until), ptrStr(in.Level), ptrStr(in.Keyword))
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		if f.Empty() {
			return nil, LogsToolOutput{}, fmt.Errorf("filter_logs requires at least one of: since, until, level, keyword")
		}
		max := clampMax(ptrInt64(in.MaxLines), defaultFilterMax)
		q := source.Query{StreamID: sid}
		lines, trunc, err := logtool.Collect(ctx, src, q, f, max)
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		return nil, toLogsOutput(sid, lines, trunc), nil
	}
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func requireLogSource(src source.LogSource) error {
	if src == nil {
		return fmt.Errorf("log access not configured: set LOG_MCP_ROOT to a readable file or directory (use OS path list separator for multiple roots)")
	}
	return nil
}

func clampMax(n, def int64) int64 {
	if n <= 0 {
		n = def
	}
	if n > maxOutputLinesCap {
		n = maxOutputLinesCap
	}
	return n
}

func filterFromStrings(since, until, level, keyword string) (logtool.Filter, error) {
	var f logtool.Filter
	var err error
	f.Since, err = parseRFC3339Optional(since)
	if err != nil {
		return f, err
	}
	f.Until, err = parseRFC3339Optional(until)
	if err != nil {
		return f, err
	}
	f.Level = strings.TrimSpace(level)
	f.Keyword = strings.TrimSpace(keyword)
	return f, nil
}

func parseRFC3339Optional(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("invalid RFC3339 time %q: %w", s, err)
	}
	return &t, nil
}

func toLogsOutput(streamID string, lines []logtool.Line, truncated bool) LogsToolOutput {
	out := LogsToolOutput{
		StreamID:  streamID,
		Lines:     make([]LogLineJSON, 0, len(lines)),
		Truncated: truncated,
	}
	for _, ln := range lines {
		j := LogLineJSON{
			LineNumber: ln.LineNumber,
			Level:      ln.Level,
			Message:    ln.Message,
			Text:       ln.Text,
		}
		if ln.Timestamp != nil {
			j.Timestamp = ln.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		out.Lines = append(out.Lines, j)
	}
	return out
}
