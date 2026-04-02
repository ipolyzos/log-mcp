package logcorr

import (
	"testing"

	"github.com/ipolyzos/log-mcp/internal/parse"
)

func TestGetField_traceIdAliases(t *testing.T) {
	t.Parallel()
	p := parse.ParsedLine{Fields: map[string]string{"traceId": "abc-123"}}
	v, ok := GetField(p, "trace_id")
	if !ok || v != "abc-123" {
		t.Fatalf("traceId: got %q %v", v, ok)
	}
	p2 := parse.ParsedLine{Fields: map[string]string{"x-trace-id": "t2"}}
	v, ok = GetField(p2, "trace_id")
	if !ok || v != "t2" {
		t.Fatalf("x-trace-id: got %q %v", v, ok)
	}
}

func TestGetField_requestIdAliases(t *testing.T) {
	t.Parallel()
	p := parse.ParsedLine{Fields: map[string]string{"req_id": "r9"}}
	v, ok := GetField(p, "request_id")
	if !ok || v != "r9" {
		t.Fatalf("req_id: got %q %v", v, ok)
	}
	p2 := parse.ParsedLine{Fields: map[string]string{"X-Request-Id": "xr"}}
	v, ok = GetField(p2, "request_id")
	if !ok || v != "xr" {
		t.Fatalf("X-Request-Id: got %q %v", v, ok)
	}
}

func TestGetField_customKey(t *testing.T) {
	t.Parallel()
	p := parse.ParsedLine{Fields: map[string]string{"tenant": "acme"}}
	v, ok := GetField(p, "tenant")
	if !ok || v != "acme" {
		t.Fatalf("custom: got %q %v", v, ok)
	}
}

func TestGetField_missing(t *testing.T) {
	t.Parallel()
	p := parse.ParsedLine{Fields: map[string]string{"msg": "x"}}
	_, ok := GetField(p, "trace_id")
	if ok {
		t.Fatal("expected missing")
	}
}
