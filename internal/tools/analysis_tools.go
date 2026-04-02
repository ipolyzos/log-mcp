package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/loganalysis"
	"github.com/ipolyzos/log-mcp/internal/source"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameDetectAnomalies is the MCP tool name for statistical anomaly hints.
const ToolNameDetectAnomalies = "detect_anomalies"

// ToolNameExtractTraceFlow is the MCP tool name for ordered trace timelines.
const ToolNameExtractTraceFlow = "extract_trace_flow"

type detectAnomaliesInput struct {
	StreamID        string   `json:"stream_id" jsonschema:"required log stream id from the configured source"`
	MaxScanLines    *int64   `json:"max_scan_lines,omitempty" jsonschema:"max lines to scan; omit for default"`
	SegmentCount    *int64   `json:"segment_count,omitempty" jsonschema:"line segments for error-rate comparison (default 15)"`
	ZScoreThreshold *float64 `json:"z_score_threshold,omitempty" jsonschema:"min z-score to flag a segment (default 2)"`
	MinRepeatErrors *int64   `json:"min_repeat_errors,omitempty" jsonschema:"min identical ERROR lines to report as burst (default 6)"`
}

// DetectAnomaliesOutput is the structured result for detect_anomalies.
type DetectAnomaliesOutput struct {
	StreamID           string                           `json:"stream_id"`
	ScannedLines       int64                            `json:"scanned_lines"`
	Truncated          bool                             `json:"truncated"`
	LowSample          bool                             `json:"low_sample"`
	LevelSpikeSegments []loganalysis.LevelSpikeSegment  `json:"level_spike_segments"`
	RepeatedErrors     []loganalysis.RepeatedErrorBurst `json:"repeated_errors"`
}

type extractTraceFlowInput struct {
	StreamID     string  `json:"stream_id" jsonschema:"required log stream id from the configured source"`
	TraceID      string  `json:"trace_id" jsonschema:"required trace id value as it appears in logs"`
	Field        *string `json:"field,omitempty" jsonschema:"logical field name (default trace_id); supports request_id aliases"`
	MaxScanLines *int64  `json:"max_scan_lines,omitempty" jsonschema:"max lines to scan; omit for default"`
	MaxSteps     *int64  `json:"max_steps,omitempty" jsonschema:"max events in the flow; omit for default"`
}

// ExtractTraceFlowOutput is the structured result for extract_trace_flow.
type ExtractTraceFlowOutput struct {
	StreamID     string                      `json:"stream_id"`
	Field        string                      `json:"field"`
	TraceID      string                      `json:"trace_id"`
	ScannedLines int64                       `json:"scanned_lines"`
	Truncated    bool                        `json:"truncated"`
	Steps        []loganalysis.TraceFlowStep `json:"steps"`
}

func registerAdvancedAnalysis(s *mcp.Server, src source.LogSource) {
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameDetectAnomalies,
		Description: "Deterministic anomaly hints (no ML): scans redacted logs, splits the stream into equal line-count segments, " +
			"flags segments where ERROR/FATAL/CRITICAL counts are unusually high vs other segments (z-score), " +
			"and lists highly repeated identical ERROR lines (fingerprint of redacted text). " +
			"When the stream is short vs segment_count or all lines fall in one segment, low_sample is true and level_spike_segments is empty (no cross-segment comparison). " +
			"Respects LOG_MCP_SEC_MAX_SCAN_LINES.",
	}, detectAnomaliesHandler(src))
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameExtractTraceFlow,
		Description: "Extract an ordered timeline for one trace_id (or request_id): same correlation as correlate_events, " +
			"but returns compact steps with timestamp, service, span_id (when present), level, message, and redacted text. " +
			"Field defaults to trace_id logical name.",
	}, extractTraceFlowHandler(src))
}

func detectAnomaliesHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, detectAnomaliesInput) (*mcp.CallToolResult, DetectAnomaliesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in detectAnomaliesInput) (*mcp.CallToolResult, DetectAnomaliesOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, DetectAnomaliesOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, DetectAnomaliesOutput{}, fmt.Errorf("stream_id is required")
		}
		opts := loganalysis.AnomalyOptions{
			MaxScanLines: ptrInt64(in.MaxScanLines),
		}
		if in.SegmentCount != nil && *in.SegmentCount >= 2 {
			opts.SegmentCount = int(*in.SegmentCount)
		}
		if in.ZScoreThreshold != nil && *in.ZScoreThreshold > 0 {
			opts.ZScoreThreshold = *in.ZScoreThreshold
		}
		if in.MinRepeatErrors != nil && *in.MinRepeatErrors > 0 {
			opts.MinRepeatErrors = *in.MinRepeatErrors
		}
		rep, err := loganalysis.DetectAnomalies(ctx, src, sid, opts)
		if err != nil {
			return nil, DetectAnomaliesOutput{}, err
		}
		return nil, DetectAnomaliesOutput{
			StreamID:           rep.StreamID,
			ScannedLines:       rep.ScannedLines,
			Truncated:          rep.Truncated,
			LowSample:          rep.LowSample,
			LevelSpikeSegments: rep.LevelSpikeSegments,
			RepeatedErrors:     rep.RepeatedErrors,
		}, nil
	}
}

func extractTraceFlowHandler(src source.LogSource) func(context.Context, *mcp.CallToolRequest, extractTraceFlowInput) (*mcp.CallToolResult, ExtractTraceFlowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in extractTraceFlowInput) (*mcp.CallToolResult, ExtractTraceFlowOutput, error) {
		if err := requireLogSource(src); err != nil {
			return nil, ExtractTraceFlowOutput{}, err
		}
		sid := strings.TrimSpace(in.StreamID)
		if sid == "" {
			return nil, ExtractTraceFlowOutput{}, fmt.Errorf("stream_id is required")
		}
		tid := strings.TrimSpace(in.TraceID)
		if tid == "" {
			return nil, ExtractTraceFlowOutput{}, fmt.Errorf("trace_id is required")
		}
		field := ""
		if in.Field != nil {
			field = strings.TrimSpace(*in.Field)
		}
		maxScan := ptrInt64(in.MaxScanLines)
		maxSteps := ptrInt64(in.MaxSteps)
		res, err := loganalysis.ExtractTraceFlow(ctx, src, sid, field, tid, maxScan, maxSteps)
		if err != nil {
			return nil, ExtractTraceFlowOutput{}, err
		}
		return nil, ExtractTraceFlowOutput{
			StreamID:     res.StreamID,
			Field:        res.Field,
			TraceID:      res.TraceID,
			ScannedLines: res.ScannedLines,
			Truncated:    res.Truncated,
			Steps:        res.Steps,
		}, nil
	}
}
