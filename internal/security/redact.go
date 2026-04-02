package security

import "github.com/ipolyzos/log-mcp/internal/normalize"

// RedactionEngine applies the canonical masking and normalization pipeline.
// It always uses [normalize.DefaultConfig]; output is not configurable to avoid
// accidentally disabling redaction at tool boundaries.
type RedactionEngine struct{}

// Apply redacts one raw log line for safe tool and aggregation output.
func (RedactionEngine) Apply(s string) string {
	return normalize.Apply(s, normalize.DefaultConfig())
}

// RedactLine is the package-level helper used by the log pipeline and tools.
func RedactLine(s string) string {
	return normalize.Apply(s, normalize.DefaultConfig())
}
