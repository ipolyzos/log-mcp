package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ipolyzos/log-mcp/internal/llmtools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolNameIncidentSummary is the MCP tool for LLM-oriented incident briefs from structured summaries only.
const ToolNameIncidentSummary = "incident_summary"

// ToolNameExplainLogPattern explains a single structured error pattern (no log reads).
const ToolNameExplainLogPattern = "explain_log_pattern"

type incidentSummaryInput struct {
	// StructuredSummary is a JSON object (summarize_errors and/or detect_anomalies output).
	StructuredSummary any `json:"structured_summary" jsonschema:"required object from summarize_errors and/or detect_anomalies; does not read logs"`
}

// IncidentSummaryOutput wraps a deterministic brief; no model inference in-process.
type IncidentSummaryOutput struct {
	Headline        string   `json:"headline"`
	Bullets         []string `json:"bullets"`
	ForLLMContext   string   `json:"for_llm_context"`
	SummaryKindHint string   `json:"summary_kind_hint"`
}

type explainLogPatternInput struct {
	ErrorGroup    *llmtools.ErrorGroupPattern    `json:"error_group,omitempty" jsonschema:"one summarize_errors group object"`
	RepeatedBurst *llmtools.RepeatedBurstPattern `json:"repeated_burst,omitempty" jsonschema:"one detect_anomalies repeated_errors entry"`
}

// ExplainLogPatternOutput is deterministic interpretation hints for an LLM.
type ExplainLogPatternOutput struct {
	PatternKind    string   `json:"pattern_kind"`
	Overview       string   `json:"overview"`
	Interpretation []string `json:"interpretation"`
	LLMPromptHints []string `json:"llm_prompt_hints"`
}

// RegisterLLMTools registers tools that only consume structured summaries (no LogSource).
func RegisterLLMTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameIncidentSummary,
		Description: "Build a deterministic incident-style briefing for an LLM from structured_summary JSON only. " +
			"Pass the object returned by summarize_errors and/or detect_anomalies (merged object is allowed). " +
			"Does not read stream_id or raw logs. Rejects read_logs-style payloads.",
	}, incidentSummaryHandler)
	mcp.AddTool(s, &mcp.Tool{
		Name: ToolNameExplainLogPattern,
		Description: "Return deterministic interpretation hints for one structured pattern only: " +
			"either error_group (one element shaped like summarize_errors.groups[]) or repeated_burst " +
			"(one element shaped like detect_anomalies.repeated_errors[]). Exactly one must be set. No log access.",
	}, explainLogPatternHandler)
}

func incidentSummaryHandler(_ context.Context, _ *mcp.CallToolRequest, in incidentSummaryInput) (*mcp.CallToolResult, IncidentSummaryOutput, error) {
	if in.StructuredSummary == nil {
		return nil, IncidentSummaryOutput{}, fmt.Errorf("structured_summary is required")
	}
	raw, err := json.Marshal(in.StructuredSummary)
	if err != nil {
		return nil, IncidentSummaryOutput{}, fmt.Errorf("structured_summary: %w", err)
	}
	obj, err := llmtools.ValidateStructuredSummary(raw)
	if err != nil {
		return nil, IncidentSummaryOutput{}, err
	}
	brief, err := llmtools.BuildIncidentBrief(obj)
	if err != nil {
		return nil, IncidentSummaryOutput{}, err
	}
	return nil, IncidentSummaryOutput{
		Headline:        brief.Headline,
		Bullets:         brief.Bullets,
		ForLLMContext:   brief.ForLLMContext,
		SummaryKindHint: brief.SummaryKindHint,
	}, nil
}

func explainLogPatternHandler(_ context.Context, _ *mcp.CallToolRequest, in explainLogPatternInput) (*mcp.CallToolResult, ExplainLogPatternOutput, error) {
	eg := in.ErrorGroup
	rb := in.RepeatedBurst
	if (eg == nil && rb == nil) || (eg != nil && rb != nil) {
		return nil, ExplainLogPatternOutput{}, fmt.Errorf("provide exactly one of error_group or repeated_burst")
	}
	var exp *llmtools.PatternExplanation
	var err error
	if eg != nil {
		exp, err = llmtools.ExplainErrorGroup(*eg)
	} else {
		exp, err = llmtools.ExplainRepeatedBurst(*rb)
	}
	if err != nil {
		return nil, ExplainLogPatternOutput{}, err
	}
	return nil, ExplainLogPatternOutput{
		PatternKind:    exp.PatternKind,
		Overview:       exp.Overview,
		Interpretation: exp.Interpretation,
		LLMPromptHints: exp.LLMPromptHints,
	}, nil
}
