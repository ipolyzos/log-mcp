// Package pipeline wires [source.LogSource] reads through parse, redaction, and
// analyze stages using bounded Go channels so lines stream through without
// accumulating the full log in memory.
package pipeline

import (
	"context"

	"github.com/ipolyzos/log-mcp/internal/parse"
	"github.com/ipolyzos/log-mcp/internal/security"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// LineParser parses one raw log line (typically [parse.AutoParser]).
type LineParser interface {
	Parse(line string) (parse.ParsedLine, error)
}

// Analyzer runs per-line logic after normalization (metrics, classification, etc.).
type Analyzer interface {
	Analyze(ctx context.Context, n NormalizedEntry) (Analysis, error)
}

// Analysis is optional output from [Analyzer]; tools may embed richer results later.
type Analysis struct {
	Level string
}

// NormalizedEntry is one line after masking/normalization with a structured view
// re-parsed from the redacted raw line so fields align with [Normalized.Redacted].
type NormalizedEntry struct {
	LineNumber int64
	Redacted   string
	Parsed     parse.ParsedLine
	// FirstParseErr is the error from the initial parse of the original raw line, if any.
	FirstParseErr error
}

// ParsedEntry carries the first parse plus original raw for the normalizer stage.
type ParsedEntry struct {
	LineNumber int64
	Raw        string
	Parsed     parse.ParsedLine
	ParseErr   error
}

// Result is the terminal payload for one line after all stages.
type Result struct {
	LineNumber int64
	Redacted   string
	Parsed     parse.ParsedLine
	ParseErr   error
	Analysis   Analysis
}

// Pipeline streams log lines from a [source.LogSource] through parser →
// [security.RedactLine] → analyzer. Channel buffers are bounded ([ChannelBuffer]); each
// stage holds at most a few in-flight values per connection.
type Pipeline struct {
	Source   source.LogSource
	Query    source.Query
	Parser   LineParser
	Analyzer Analyzer
	// ChannelBuffer is the per-stage channel capacity; zero or negative defaults to 32.
	ChannelBuffer int
}

// Run starts source.Read and connects four stages with bounded channels. The
// returned channel is closed after the source channel closes and the analyzer
// finishes. The caller must read until closed. If Read fails, err is non-nil
// and the result channel is nil.
func (p *Pipeline) Run(ctx context.Context) (<-chan Result, error) {
	entries, err := p.Source.Read(ctx, p.Query)
	if err != nil {
		return nil, err
	}
	buf := p.ChannelBuffer
	if buf <= 0 {
		buf = 32
	}
	parser := p.Parser
	if parser == nil {
		parser = parse.AutoParser{}
	}
	analyzer := p.Analyzer

	chParsed := make(chan ParsedEntry, buf)
	chNorm := make(chan NormalizedEntry, buf)
	out := make(chan Result, buf)

	go func() {
		defer close(chParsed)
		for e := range entries {
			pl, perr := parser.Parse(e.Raw)
			chParsed <- ParsedEntry{
				LineNumber: e.LineNumber,
				Raw:        e.Raw,
				Parsed:     pl,
				ParseErr:   perr,
			}
		}
	}()

	go func() {
		defer close(chNorm)
		for pe := range chParsed {
			red := security.RedactLine(pe.Raw)
			safeParsed, _ := parser.Parse(red)
			chNorm <- NormalizedEntry{
				LineNumber:    pe.LineNumber,
				Redacted:      red,
				Parsed:        safeParsed,
				FirstParseErr: pe.ParseErr,
			}
		}
	}()

	go func() {
		defer close(out)
		for ne := range chNorm {
			var a Analysis
			if analyzer != nil {
				a, _ = analyzer.Analyze(ctx, ne)
			}
			out <- Result{
				LineNumber: ne.LineNumber,
				Redacted:   ne.Redacted,
				Parsed:     ne.Parsed,
				ParseErr:   ne.FirstParseErr,
				Analysis:   a,
			}
		}
	}()

	return out, nil
}
