package logtool

import (
	"strings"
	"time"

	"github.com/ipolyzos/log-mcp/internal/pipeline"
)

// Filter limits which pipeline results are kept (AND semantics). Zero value
// matches every line.
type Filter struct {
	Since   *time.Time
	Until   *time.Time
	Level   string
	Keyword string
}

// Empty reports whether no filter criteria are set.
func (f Filter) Empty() bool {
	return f.Since == nil && f.Until == nil && f.Level == "" && f.Keyword == ""
}

// Match returns whether r passes all active filter criteria.
func Match(f Filter, r pipeline.Result) bool {
	if f.Keyword != "" {
		kw := strings.ToLower(f.Keyword)
		if !strings.Contains(strings.ToLower(r.Redacted), kw) {
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
