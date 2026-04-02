// Package errsummary implements deterministic error grouping and ranking over
// parsed log lines (no LLM). Used by the summarize_errors MCP tool.
package errsummary

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/logtool"
	"github.com/ipolyzos/log-mcp/internal/parse"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/security"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// Options configures scanning bounds and stream metadata for service fallback.
type Options struct {
	// MaxScanLines is passed to [source.Query.MaxLines]; 0 defaults to 20000. Capped by [security.LimitsFromEnv].MaxScanLines.
	MaxScanLines int64
	// StreamID is used for output and for filepath.Base fallback service label.
	StreamID string
}

// Summary is the aggregated result of summarization.
type Summary struct {
	StreamID       string
	ScannedLines   int64
	ErrorLineCount int64
	Groups         []Group
	Truncated      bool
}

// Group is one fingerprint bucket with frequency and per-service counts.
type Group struct {
	Rank        int
	Fingerprint string
	Pattern     string
	Count       int64
	ByService   []ServiceCount
}

// ServiceCount is a stable (sorted) service → count pair for JSON output.
type ServiceCount struct {
	Service string `json:"service"`
	Count   int64  `json:"count"`
}

var spaceSquash = regexp.MustCompile(`\s+`)

var extraSignatureKeys = []string{
	"code", "err", "error", "error_code", "error.type", "exception", "type",
}

var serviceFieldKeys = []string{
	"service", "service_name", "servicename", "app", "application",
	"logger", "component", "pod", "container",
}

// Summarize streams up to MaxScanLines through the redaction pipeline, keeps
// lines whose level looks like an error, and groups them by a deterministic
// fingerprint. Groups are sorted by count descending, then fingerprint ascending.
func Summarize(ctx context.Context, src source.LogSource, opts Options) (*Summary, error) {
	maxScan := opts.MaxScanLines
	if maxScan <= 0 {
		maxScan = 20000
	}
	lim := security.LimitsFromEnv()
	if maxScan > lim.MaxScanLines {
		maxScan = lim.MaxScanLines
	}
	streamBase := filepath.Base(strings.TrimSpace(opts.StreamID))
	if streamBase == "" || streamBase == "." {
		streamBase = "<stream>"
	}

	q := source.Query{
		StreamID: strings.TrimSpace(opts.StreamID),
		MaxLines: maxScan,
	}
	logtool.EnrichQuery(&q)
	pl := &pipeline.Pipeline{
		Source: src,
		Query:  q,
	}
	out, err := pl.Run(ctx)
	if err != nil {
		return nil, err
	}

	type acc struct {
		count   int64
		pattern string
		bySvc   map[string]int64
	}
	groups := make(map[string]*acc)

	var scanned, errLines int64
	for r := range out {
		scanned++
		if !isErrorLevel(r.Parsed.Level) {
			continue
		}
		errLines++
		raw := rawSignature(r.Parsed, r.Redacted)
		fp := Fingerprint(raw)
		a, ok := groups[fp]
		if !ok {
			a = &acc{bySvc: make(map[string]int64)}
			a.pattern = displayPattern(raw)
			groups[fp] = a
		}
		a.count++
		svc := serviceLabel(r.Parsed.Fields, streamBase)
		a.bySvc[svc]++
	}

	type pair struct {
		fp  string
		acc *acc
	}
	list := make([]pair, 0, len(groups))
	for fp, a := range groups {
		list = append(list, pair{fp: fp, acc: a})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].acc.count != list[j].acc.count {
			return list[i].acc.count > list[j].acc.count
		}
		return list[i].fp < list[j].fp
	})

	outGroups := make([]Group, 0, len(list))
	for i, p := range list {
		svcPairs := make([]ServiceCount, 0, len(p.acc.bySvc))
		for s, c := range p.acc.bySvc {
			svcPairs = append(svcPairs, ServiceCount{Service: s, Count: c})
		}
		sort.Slice(svcPairs, func(a, b int) bool {
			if svcPairs[a].Count != svcPairs[b].Count {
				return svcPairs[a].Count > svcPairs[b].Count
			}
			return svcPairs[a].Service < svcPairs[b].Service
		})
		outGroups = append(outGroups, Group{
			Rank:        i + 1,
			Fingerprint: p.fp,
			Pattern:     p.acc.pattern,
			Count:       p.acc.count,
			ByService:   svcPairs,
		})
	}

	truncated := maxScan > 0 && scanned >= maxScan
	return &Summary{
		StreamID:       opts.StreamID,
		ScannedLines:   scanned,
		ErrorLineCount: errLines,
		Groups:         outGroups,
		Truncated:      truncated,
	}, nil
}

// Fingerprint returns the stable grouping key for a raw signature string.
func Fingerprint(rawSignature string) string {
	s := strings.TrimSpace(rawSignature)
	if s == "" {
		s = "<empty>"
	}
	s = security.RedactLine(s)
	s = strings.ToLower(s)
	return strings.TrimSpace(spaceSquash.ReplaceAllString(s, " "))
}

func rawSignature(p parse.ParsedLine, redactedFallback string) string {
	msg := strings.TrimSpace(p.Message)
	if msg == "" {
		msg = strings.TrimSpace(redactedFallback)
	}
	lower := lowerFieldMap(p.Fields)
	var extras []string
	for _, k := range extraSignatureKeys {
		if v := lower[k]; v != "" {
			extras = append(extras, k+"="+v)
		}
	}
	sort.Strings(extras)
	if len(extras) > 0 {
		return msg + "\x1e" + strings.Join(extras, "\x1e")
	}
	return msg
}

func displayPattern(rawSig string) string {
	// Show first line of signature (before record separator) with redaction, no lowercasing.
	part := rawSig
	if i := strings.IndexByte(rawSig, '\x1e'); i >= 0 {
		part = rawSig[:i]
	}
	s := security.RedactLine(strings.TrimSpace(part))
	if s == "" {
		s = "<empty>"
	}
	if len(s) > 280 {
		s = s[:280] + "…"
	}
	return s
}

func lowerFieldMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	o := make(map[string]string, len(m))
	for k, v := range m {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		o[strings.ToLower(k)] = v
	}
	return o
}

func serviceLabel(fields map[string]string, streamBase string) string {
	lower := lowerFieldMap(fields)
	for _, k := range serviceFieldKeys {
		if v := lower[k]; v != "" {
			return v
		}
	}
	return streamBase
}

func isErrorLevel(lvl string) bool {
	switch strings.ToUpper(strings.TrimSpace(lvl)) {
	case "ERROR", "ERR", "FATAL", "CRITICAL", "PANIC", "SEVERE":
		return true
	default:
		return false
	}
}
