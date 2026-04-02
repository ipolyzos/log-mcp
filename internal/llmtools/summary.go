// Package llmtools builds deterministic reasoning briefs from structured log tool
// outputs only (no raw log access).
package llmtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// MaxStructuredSummaryBytes caps the JSON payload for incident-style tools.
const MaxStructuredSummaryBytes = 512 * 1024

// ErrNotStructuredSummary is returned when input is not an allowed summary object.
var ErrNotStructuredSummary = fmt.Errorf("llmtools: payload is not a recognized structured summary (need summarize_errors-style groups and/or detect_anomalies-style segments)")

// ValidateStructuredSummary checks that b is a JSON object within size limits and
// contains fields produced by summarize_errors and/or detect_anomalies (not raw log lines).
func ValidateStructuredSummary(b []byte) (map[string]json.RawMessage, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, fmt.Errorf("llmtools: structured_summary is empty")
	}
	if len(b) > MaxStructuredSummaryBytes {
		return nil, fmt.Errorf("llmtools: structured_summary exceeds max size (%d bytes)", MaxStructuredSummaryBytes)
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("llmtools: structured_summary is not valid JSON")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("llmtools: structured_summary must be a JSON object: %w", err)
	}
	if len(obj) == 0 {
		return nil, ErrNotStructuredSummary
	}
	// Reject obvious raw log tool payloads (line-oriented reads).
	if _, ok := obj["lines"]; ok {
		if _, hasGroups := obj["groups"]; !hasGroups {
			return nil, fmt.Errorf("llmtools: raw read_logs-style payload is not allowed; pass summarize_errors or detect_anomalies output only")
		}
	}
	hasErrSummary := false
	if raw, ok := obj["groups"]; ok {
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) == nil {
			hasErrSummary = true
		}
	}
	hasAnomaly := false
	if raw, ok := obj["level_spike_segments"]; ok {
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) == nil {
			hasAnomaly = true
		}
	}
	if raw, ok := obj["repeated_errors"]; ok {
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) == nil {
			hasAnomaly = true
		}
	}
	if !hasErrSummary && !hasAnomaly {
		return nil, ErrNotStructuredSummary
	}
	return obj, nil
}

// IncidentBrief is deterministic text for downstream LLM consumption.
type IncidentBrief struct {
	Headline        string   `json:"headline"`
	Bullets         []string `json:"bullets"`
	ForLLMContext   string   `json:"for_llm_context"`
	SummaryKindHint string   `json:"summary_kind_hint"`
}

// BuildIncidentBrief renders a short brief from validated summary JSON (object map).
func BuildIncidentBrief(obj map[string]json.RawMessage) (*IncidentBrief, error) {
	var se summarizeErrorsShape
	_ = json.Unmarshal(mustRaw(obj, "groups"), &se.Groups)
	_ = json.Unmarshal(obj["error_line_count"], &se.ErrorLineCount)
	_ = json.Unmarshal(obj["scanned_lines"], &se.ScannedLines)
	_ = json.Unmarshal(obj["truncated"], &se.Truncated)

	var da detectAnomaliesShape
	_ = json.Unmarshal(mustRaw(obj, "level_spike_segments"), &da.LevelSpikeSegments)
	_ = json.Unmarshal(mustRaw(obj, "repeated_errors"), &da.RepeatedErrors)
	if raw, ok := obj["repeated_warn_bursts"]; ok {
		_ = json.Unmarshal(raw, &da.RepeatedWarnBursts)
	}
	_ = json.Unmarshal(obj["scanned_lines"], &da.ScannedLines)
	_ = json.Unmarshal(obj["truncated"], &da.Truncated)
	if raw, ok := obj["low_sample"]; ok {
		_ = json.Unmarshal(raw, &da.LowSample)
	}

	kind := make([]string, 0, 3)
	if _, ok := obj["groups"]; ok {
		kind = append(kind, "summarize_errors")
	}
	if _, ok := obj["level_spike_segments"]; ok {
		kind = append(kind, "level_spike_segments")
	}
	if _, ok := obj["repeated_errors"]; ok {
		kind = append(kind, "repeated_errors")
	}
	if _, ok := obj["repeated_warn_bursts"]; ok {
		kind = append(kind, "repeated_warn_bursts")
	}
	if len(kind) == 0 {
		kind = append(kind, "partial")
	}

	var bullets []string
	if da.LowSample {
		bullets = append(bullets, "Anomaly inputs are low_sample: segment z-score spikes were not computed (short stream vs segment_count or fewer than two active segments); treat geographic spike claims cautiously and rely on repeated_errors, repeated_warn_bursts, or summarize_errors when present.")
	}
	headline := "Incident summary (from structured tools only)"
	if se.ErrorLineCount > 0 {
		headline = fmt.Sprintf("Error digest: %d error-level lines referenced in summary", se.ErrorLineCount)
	}
	if len(se.Groups) > 0 {
		top := se.Groups[0]
		bullets = append(bullets, fmt.Sprintf("Top error group #%d: %d occurrences — pattern: %s", top.Rank, top.Count, truncate(top.Pattern, 120)))
		for i := 1; i < len(se.Groups) && i < 5; i++ {
			g := se.Groups[i]
			bullets = append(bullets, fmt.Sprintf("Group #%d: %d× — %s", g.Rank, g.Count, truncate(g.Pattern, 100)))
		}
	}
	for _, s := range da.LevelSpikeSegments {
		if s.ErrorCount == 0 {
			continue
		}
		bullets = append(bullets, fmt.Sprintf("Error-rate spike in segment %d (lines %d–%d): %d errors, z≈%.2f vs segment baseline", s.SegmentIndex, s.LineStart, s.LineEnd, s.ErrorCount, s.ZScore))
	}
	for _, r := range da.RepeatedErrors {
		if r.Count == 0 {
			continue
		}
		bullets = append(bullets, fmt.Sprintf("Repeated identical ERROR×%d (lines %d–%d): %s", r.Count, r.FirstLine, r.LastLine, truncate(r.SampleMessage, 100)))
	}
	for _, r := range da.RepeatedWarnBursts {
		if r.Count == 0 {
			continue
		}
		bullets = append(bullets, fmt.Sprintf("Repeated identical WARN×%d (lines %d–%d): %s", r.Count, r.FirstLine, r.LastLine, truncate(r.SampleMessage, 100)))
	}
	if len(bullets) == 0 {
		bullets = append(bullets, "No ranked error groups or anomaly segments in this summary (may be empty scan or clean window).")
	}

	var ctx strings.Builder
	ctx.WriteString("Use this briefing together with the original structured_summary JSON. ")
	ctx.WriteString("Do not treat this text as ground truth beyond what the summary objects state. ")
	if se.Truncated || da.Truncated {
		ctx.WriteString("Source summaries were truncated; conclusions are incomplete. ")
	}
	if da.LowSample {
		ctx.WriteString("detect_anomalies reported low_sample; do not infer segment spike geography from omitted z-scores. ")
	}
	ctx.WriteString("Prefer correlating trace_id/request_id using correlate_events if IDs appear in patterns.")

	return &IncidentBrief{
		Headline:        headline,
		Bullets:         bullets,
		ForLLMContext:   ctx.String(),
		SummaryKindHint: strings.Join(kind, "+"),
	}, nil
}

func mustRaw(obj map[string]json.RawMessage, key string) json.RawMessage {
	if v, ok := obj[key]; ok {
		return v
	}
	return json.RawMessage("null")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

type summarizeErrorsShape struct {
	Groups         []groupShape `json:"groups"`
	ErrorLineCount int64        `json:"error_line_count"`
	ScannedLines   int64        `json:"scanned_lines"`
	Truncated      bool         `json:"truncated"`
}

type groupShape struct {
	Rank    int    `json:"rank"`
	Count   int64  `json:"count"`
	Pattern string `json:"pattern"`
}

type detectAnomaliesShape struct {
	LevelSpikeSegments []spikeShape `json:"level_spike_segments"`
	RepeatedErrors     []burstShape `json:"repeated_errors"`
	RepeatedWarnBursts []burstShape `json:"repeated_warn_bursts"`
	ScannedLines       int64        `json:"scanned_lines"`
	Truncated          bool         `json:"truncated"`
	LowSample          bool         `json:"low_sample"`
}

type spikeShape struct {
	SegmentIndex int     `json:"segment_index"`
	LineStart    int64   `json:"line_start"`
	LineEnd      int64   `json:"line_end"`
	ErrorCount   int64   `json:"error_count"`
	ZScore       float64 `json:"z_score"`
}

type burstShape struct {
	SampleMessage string `json:"sample_message"`
	Count         int64  `json:"count"`
	FirstLine     int64  `json:"first_line"`
	LastLine      int64  `json:"last_line"`
}
