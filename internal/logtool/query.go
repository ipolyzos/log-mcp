package logtool

import (
	"github.com/ipolyzos/log-mcp/internal/security"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// EnrichQuery applies process-wide I/O caps from [security.LimitsFromEnv] when
// MaxBytes / MaxLineBytes are unset (zero).
func EnrichQuery(q *source.Query) {
	lim := security.LimitsFromEnv()
	if q.MaxBytes <= 0 && lim.MaxStreamBytes > 0 {
		q.MaxBytes = lim.MaxStreamBytes
	}
	if q.MaxLineBytes <= 0 && lim.MaxLineBytes > 0 {
		q.MaxLineBytes = lim.MaxLineBytes
	}
}
