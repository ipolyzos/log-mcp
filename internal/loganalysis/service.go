// Package loganalysis provides deterministic anomaly detection and trace-flow extraction
// on redacted pipeline output (no ML).
package loganalysis

import (
	"strings"

	"github.com/ipolyzos/log-mcp/internal/parse"
)

var serviceFieldKeys = []string{
	"service", "service_name", "servicename", "app", "application",
	"logger", "component", "pod", "container",
}

var spanFieldKeys = []string{
	"span_id", "spanId", "dd.span_id", "otel.span_id",
}

// ServiceFromFields picks a display service name from parsed JSON fields.
func ServiceFromFields(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	lower := make(map[string]string, len(fields))
	for k, v := range fields {
		lower[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	for _, k := range serviceFieldKeys {
		if v := lower[k]; v != "" {
			return v
		}
	}
	return ""
}

// SpanIDFromFields returns a span identifier when present.
func SpanIDFromFields(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	lower := make(map[string]string, len(fields))
	for k, v := range fields {
		lower[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	for _, k := range spanFieldKeys {
		if v := lower[strings.ToLower(strings.TrimSpace(k))]; v != "" {
			return v
		}
	}
	return ""
}

func sampleMessage(p parse.ParsedLine, redacted string) string {
	s := strings.TrimSpace(p.Message)
	if s == "" {
		s = strings.TrimSpace(redacted)
	}
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
