package loganalysis

import (
	"context"
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/logcorr"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// TraceFlowResult is an ordered list of steps for one trace (or other correlated id).
type TraceFlowResult struct {
	StreamID     string
	Field        string
	TraceID      string
	ScannedLines int64
	Truncated    bool
	Steps        []TraceFlowStep
}

// TraceFlowStep is one event in chronological order for the trace.
type TraceFlowStep struct {
	Step       int    `json:"step"`
	LineNumber int64  `json:"line_number"`
	Timestamp  string `json:"timestamp,omitempty"`
	Service    string `json:"service,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	Level      string `json:"level,omitempty"`
	Message    string `json:"message,omitempty"`
	Text       string `json:"text"`
}

// ExtractTraceFlow returns redacted events for a single trace id (or other field value),
// ordered by timestamp then line number, with service/span hints from JSON fields.
func ExtractTraceFlow(ctx context.Context, src source.LogSource, streamID, field, traceValue string, maxScan, maxSteps int64) (*TraceFlowResult, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		field = "trace_id"
	}
	res, err := logcorr.CorrelateEvents(ctx, src, streamID, field, traceValue, maxScan, maxSteps)
	if err != nil {
		return nil, err
	}
	steps := make([]TraceFlowStep, 0, len(res.Events))
	for i, ev := range res.Events {
		r := ev.Result
		svc := ServiceFromFields(r.Parsed.Fields)
		span := SpanIDFromFields(r.Parsed.Fields)
		step := TraceFlowStep{
			Step:       i + 1,
			LineNumber: r.LineNumber,
			Service:    svc,
			SpanID:     span,
			Level:      r.Parsed.Level,
			Message:    r.Parsed.Message,
			Text:       r.Redacted,
		}
		if r.Parsed.Timestamp != nil {
			step.Timestamp = r.Parsed.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		steps = append(steps, step)
	}
	return &TraceFlowResult{
		StreamID:     res.StreamID,
		Field:        res.Field,
		TraceID:      res.Value,
		ScannedLines: res.ScannedLines,
		Truncated:    res.Truncated,
		Steps:        steps,
	}, nil
}
