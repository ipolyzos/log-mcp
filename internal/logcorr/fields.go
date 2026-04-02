// Package logcorr provides deterministic grouping and correlation over parsed log
// fields (e.g. trace_id, request_id) on redacted pipeline output.
package logcorr

import (
	"strings"

	"github.com/ipolyzos/log-mcp/internal/parse"
)

// FieldKeyVariants returns JSON keys to try for a logical field name.
// trace_id and request_id include common observability aliases.
func FieldKeyVariants(canonical string) []string {
	c := strings.ToLower(strings.TrimSpace(canonical))
	switch c {
	case "trace_id", "traceid":
		return []string{
			"trace_id", "traceId", "trace.id", "x-trace-id",
			"dd.trace_id", "otel.trace_id", "logging.googleapis.com/trace",
		}
	case "request_id", "requestid":
		return []string{
			"request_id", "requestId", "req_id", "correlation_id",
			"request.id", "x-request-id", "x_correlation_id",
		}
	default:
		return []string{strings.TrimSpace(canonical)}
	}
}

// GetField returns the first non-empty value for the logical field from parsed fields.
func GetField(p parse.ParsedLine, canonical string) (value string, ok bool) {
	fields := p.Fields
	if len(fields) == 0 {
		return "", false
	}
	lower := make(map[string]string, len(fields))
	for k, v := range fields {
		lower[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	for _, key := range FieldKeyVariants(canonical) {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if v, ok := fields[key]; ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v, true
			}
		}
		if v, ok := lower[strings.ToLower(key)]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}
