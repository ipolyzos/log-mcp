package llmtools

import (
	"fmt"
	"strings"
)

// ErrorGroupPattern is the structured slice of summarize_errors groups (one entry).
type ErrorGroupPattern struct {
	Rank        int    `json:"rank"`
	Count       int64  `json:"count"`
	Fingerprint string `json:"fingerprint"`
	Pattern     string `json:"pattern"`
}

// RepeatedBurstPattern matches detect_anomalies repeated_errors entries.
type RepeatedBurstPattern struct {
	Fingerprint   string `json:"fingerprint"`
	SampleMessage string `json:"sample_message"`
	Count         int64  `json:"count"`
	FirstLine     int64  `json:"first_line"`
	LastLine      int64  `json:"last_line"`
}

// PatternExplanation is deterministic guidance for an LLM about one pattern.
type PatternExplanation struct {
	PatternKind    string   `json:"pattern_kind"`
	Overview       string   `json:"overview"`
	Interpretation []string `json:"interpretation"`
	LLMPromptHints []string `json:"llm_prompt_hints"`
}

// ExplainErrorGroup builds explanation text from a summarize_errors group object.
func ExplainErrorGroup(g ErrorGroupPattern) (*PatternExplanation, error) {
	if g.Count < 0 {
		return nil, fmt.Errorf("llmtools: error_group.count must be non-negative")
	}
	if g.Count > 0 && strings.TrimSpace(g.Pattern) == "" && strings.TrimSpace(g.Fingerprint) == "" {
		return nil, fmt.Errorf("llmtools: error_group must include pattern or fingerprint when count > 0")
	}
	if g.Count == 0 && strings.TrimSpace(g.Pattern) == "" && strings.TrimSpace(g.Fingerprint) == "" {
		return nil, fmt.Errorf("llmtools: error_group is empty")
	}
	ov := fmt.Sprintf("Ranked error fingerprint #%d with %d matching redacted lines in the summarized window.", g.Rank, g.Count)
	interp := []string{
		"High count suggests a recurring failure mode or noisy dependency rather than a one-off.",
		"The pattern string is already redacted; do not infer secrets from placeholders like <token>.",
	}
	if g.Fingerprint != "" {
		interp = append(interp, "Fingerprint is stable for this server run — use it to cross-reference with other tools.")
	}
	hints := []string{
		"Ask whether the spike aligns with a deploy, dependency outage, or traffic pattern.",
		"If trace_id appears inside pattern text, use correlate_events on that id.",
	}
	return &PatternExplanation{
		PatternKind:    "summarize_errors.group",
		Overview:       ov,
		Interpretation: interp,
		LLMPromptHints: hints,
	}, nil
}

// ExplainRepeatedBurst builds explanation from a detect_anomalies repeated_errors entry.
func ExplainRepeatedBurst(b RepeatedBurstPattern) (*PatternExplanation, error) {
	if b.Count <= 0 {
		return nil, fmt.Errorf("llmtools: repeated_burst.count must be positive")
	}
	ov := fmt.Sprintf("Identical redacted ERROR line repeated %d times between lines %d and %d.", b.Count, b.FirstLine, b.LastLine)
	interp := []string{
		"Mechanical repetition usually indicates retry loops, health-check failures, or a single code path logging the same error.",
		"Compare timestamps (if you have them elsewhere) to see if bursts cluster in time.",
	}
	hints := []string{
		"Correlate with service names from surrounding structured logs if available.",
		"Check whether fingerprint matches a known deployment or config change window.",
	}
	if strings.TrimSpace(b.SampleMessage) != "" {
		interp = append(interp, "Sample message is truncated and redacted — use it only as a semantic hint.")
	}
	return &PatternExplanation{
		PatternKind:    "detect_anomalies.repeated_errors",
		Overview:       ov,
		Interpretation: interp,
		LLMPromptHints: hints,
	}, nil
}
