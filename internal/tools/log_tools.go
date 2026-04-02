package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/logtool"
	"github.com/ipolyzos/log-mcp/internal/security"
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
	defaultReadMax   int64 = 500
	defaultSearchMax int64 = 500
	defaultFilterMax int64 = 500
)

// readLogsInput is the payload for read_logs.
type readLogsInput struct {
	StreamID    string   `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	MaxLines    *int64   `json:"max_lines,omitempty" jsonschema:"max lines to return; omit for default"`
	Since       *string  `json:"since,omitempty" jsonschema:"optional RFC3339 lower time bound (inclusive)"`
	Until       *string  `json:"until,omitempty" jsonschema:"optional RFC3339 upper time bound (inclusive)"`
	Level       *string  `json:"level,omitempty" jsonschema:"optional log level e.g. INFO WARN ERROR"`
	Keyword     *string  `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring on redacted line"`
	KeywordsAny []string `json:"keywords_any,omitempty" jsonschema:"optional OR list: line must contain at least one term (case-insensitive); mutually exclusive with keyword"`
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
	StreamID    string   `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	MaxLines    *int64   `json:"max_lines,omitempty" jsonschema:"max matching lines to return; omit for default"`
	Since       *string  `json:"since,omitempty" jsonschema:"optional RFC3339 lower time bound (inclusive); ANDed with until/level/keyword when set"`
	Until       *string  `json:"until,omitempty" jsonschema:"optional RFC3339 upper time bound (inclusive); ANDed with since/level/keyword when set"`
	Level       *string  `json:"level,omitempty" jsonschema:"optional log level e.g. INFO WARN ERROR; ANDed with time range and keyword when set"`
	Keyword     *string  `json:"keyword,omitempty" jsonschema:"optional case-insensitive substring on redacted line; ANDed with time range and level when set; mutually exclusive with keywords_any"`
	KeywordsAny []string `json:"keywords_any,omitempty" jsonschema:"optional OR list on redacted line; ANDed with time range and level; mutually exclusive with keyword"`
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
			Description: "Read log lines from a stream with optional limits and filters (time range, level, keyword or keywords_any OR list). Lines are redacted. Use stream_id from the source (e.g. absolute file path under LOG_MCP_ROOT). When no filters are set, reads sequentially up to max_lines without scanning the whole file.",
		}, readLogsHandler(src))
		mcp.AddTool(s, &mcp.Tool{
			Name:        ToolNameSearchLogs,
			Description: "Search redacted log lines for a keyword (case-insensitive), with optional time range and level filters. Returns up to max_matches hits.",
		}, searchLogsHandler(src))
		mcp.AddTool(s, &mcp.Tool{
			Name:        ToolNameFilterLogs,
			Description: "Return redacted log lines where every supplied predicate matches (boolean AND across time, level, and text). You must set at least one of: since, until, level, keyword, keywords_any. Use keywords_any for OR across substrings on the redacted line (mutually exclusive with keyword in one call). Example: level=ERROR and keyword=timeout keeps ERROR lines containing timeout; level=INFO and keywords_any=[\"a\",\"b\"] keeps INFO lines containing a or b.",
		}, filterLogsHandler(src))
		registerSummarizeErrors(s, src)
		registerCorrelationTools(s, src)
		registerAdvancedAnalysis(s, src)
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
		f, err := buildFilter(ptrStr(in.Since), ptrStr(in.Until), ptrStr(in.Level), ptrStr(in.Keyword), in.KeywordsAny)
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
		f, err := buildFilter(ptrStr(in.Since), ptrStr(in.Until), ptrStr(in.Level), ptrStr(in.Keyword), in.KeywordsAny)
		if err != nil {
			return nil, LogsToolOutput{}, err
		}
		if f.Empty() {
			return nil, LogsToolOutput{}, fmt.Errorf("filter_logs requires at least one of: since, until, level, keyword, keywords_any")
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
	capN := security.LimitsFromEnv().MaxToolOutputLines
	if n > capN {
		n = capN
	}
	return n
}

const (
	maxKeywordsAny  = 32
	maxKeywordBytes = 256
)

func filterFromStrings(since, until, level, keyword string) (logtool.Filter, error) {
	return buildFilter(since, until, level, keyword, nil)
}

func buildFilter(since, until, level, keyword string, keywordsAny []string) (logtool.Filter, error) {
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
	kw := strings.TrimSpace(keyword)
	anyNorm := normalizeKeywordsAny(keywordsAny)
	if kw != "" && len(anyNorm) > 0 {
		return f, fmt.Errorf("use only one of keyword and keywords_any")
	}
	f.Keyword = kw
	f.KeywordsAny = anyNorm
	return f, nil
}

func normalizeKeywordsAny(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" || len(x) > maxKeywordBytes {
			continue
		}
		out = append(out, x)
		if len(out) >= maxKeywordsAny {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
