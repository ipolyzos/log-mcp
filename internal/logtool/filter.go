package logtool

import (
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/pipeline"
)

// Filter limits which pipeline results are kept (AND semantics across time,
// level, and text predicates). Zero value matches every line.
//
// Text matching: when KeywordsAny is non-empty, the redacted line must contain
// at least one entry (OR). When KeywordsAny is empty and Keyword is non-empty,
// the line must contain Keyword. Keyword and KeywordsAny are not both used in
// the same filter by MCP handlers (mutually exclusive at the tool layer).
type Filter struct {
	Since       *time.Time
	Until       *time.Time
	Level       string
	Keyword     string
	KeywordsAny []string
}

// Empty reports whether no filter criteria are set.
func (f Filter) Empty() bool {
	return f.Since == nil && f.Until == nil && f.Level == "" && f.Keyword == "" && len(f.KeywordsAny) == 0
}

// Match returns whether r passes all active filter criteria.
func Match(f Filter, r pipeline.Result) bool {
	lowLine := strings.ToLower(r.Redacted)
	if len(f.KeywordsAny) > 0 {
		matched := false
		for _, k := range f.KeywordsAny {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if strings.Contains(lowLine, strings.ToLower(k)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	} else if f.Keyword != "" {
		kw := strings.ToLower(f.Keyword)
		if !strings.Contains(lowLine, kw) {
			return false
		}
	}
	if f.Level != "" {
		if !strings.EqualFold(strings.TrimSpace(f.Level), strings.TrimSpace(r.Parsed.Level)) {
			return false
		}
	}
	if f.Since != nil || f.Until != nil {
		ts := r.Parsed.Timestamp
		if ts == nil {
			return false
		}
		if f.Since != nil && ts.Before(*f.Since) {
			return false
		}
		if f.Until != nil && ts.After(*f.Until) {
			return false
		}
	}
	return true
}
