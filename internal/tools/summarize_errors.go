package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/errsummary"
	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameSummarizeErrors is the MCP tools/call name for error summarization.
const ToolNameSummarizeErrors = "summarize_errors"

const (
	defaultSummarizeScan int64 = 20000
	maxSummarizeScanCap  int64 = 100000
)

// summarizeErrorsInput is the payload for summarize_errors.
type summarizeErrorsInput struct {
	StreamID     string `json:"stream_id" jsonschema:"required log stream id (path) from the configured source"`
	MaxScanLines *int64 `json:"max_scan_lines,omitempty" jsonschema:"max physical lines to scan from start of stream; omit for default"`
}

// SummarizeErrorsOutput is the structured result for summarize_errors.
type SummarizeErrorsOutput struct {
	StreamID       string               `json:"stream_id"`
	ScannedLines   int64                `json:"scanned_lines"`
	ErrorLineCount int64                `json:"error_line_count"`
	Truncated      bool                 `json:"truncated"`
	Groups         []SummarizeGroupJSON `json:"groups"`
}

// SummarizeGroupJSON is one ranked error group in [SummarizeErrorsOutput].
type SummarizeGroupJSON struct {
	Rank        int                       `json:"rank"`
	Count       int64                     `json:"count"`
	Fingerprint string                    `json:"fingerprint"`
	Pattern     string                    `json:"pattern"`
	ByService   []errsummary.ServiceCount `json:"by_service"`
}

func registerSummarizeErrors(s *mcp.Server, src source.LogSource) {
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameSummarizeErrors,
		Description: "Deterministic error summarization (no LLM): scans redacted log lines, keeps ERROR/FATAL/CRITICAL-level rows, " +
			"groups by a stable fingerprint (normalized message + error fields), ranks by frequency, and aggregates counts per service " +
			"(JSON fields service/app/logger/etc., else stream file name). Uses max_scan_lines cap (default 20000, max 100000).",
	}, summarizeErrorsHandler(src))
}

func summarizeErrorsHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, summarizeErrorsInput) (*mcp.CallToolResult, SummarizeErrorsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in summarizeErrorsInput) (*mcp.CallToolResult, SummarizeErrorsOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, SummarizeErrorsOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, SummarizeErrorsOutput{}, fmt.Errorf("stream_id is required")
		}
		maxScan := ptrInt64(in.MaxScanLines)
		if maxScan <= 0 {
			maxScan = defaultSummarizeScan
		}
		if maxScan > maxSummarizeScanCap {
			maxScan = maxSummarizeScanCap
		}
		sum, err := errsummary.Summarize(ctx, src, errsummary.Options{
			StreamID:     sid,
			MaxScanLines: maxScan,
		})
		if err != nil {
			return nil, SummarizeErrorsOutput{}, err
		}
		groups := make([]SummarizeGroupJSON, 0, len(sum.Groups))
		for _, g := range sum.Groups {
			groups = append(groups, SummarizeGroupJSON{
				Rank:        g.Rank,
				Count:       g.Count,
				Fingerprint: g.Fingerprint,
				Pattern:     g.Pattern,
				ByService:   g.ByService,
			})
		}
		return nil, SummarizeErrorsOutput{
			StreamID:       sum.StreamID,
			ScannedLines:   sum.ScannedLines,
			ErrorLineCount: sum.ErrorLineCount,
			Truncated:      sum.Truncated,
			Groups:         groups,
		}, nil
	}
}
