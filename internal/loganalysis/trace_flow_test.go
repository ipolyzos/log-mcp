package loganalysis

import (
	"context"
	"testing"
	"time"
)

func TestExtractTraceFlow_orderAndFields(t *testing.T) {
	t.Parallel()
	t1a := time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC).Format(time.RFC3339Nano)
	t1b := time.Date(2020, 1, 2, 15, 4, 6, 0, time.UTC).Format(time.RFC3339Nano)
	src := &sliceSource{
		id: "s",
		lines: []string{
			`{"level":"INFO","msg":"second","trace_id":"T1","service":"api","@timestamp":"` + t1b + `"}`,
			`{"level":"INFO","msg":"first","trace_id":"T1","service":"api","span_id":"s1","@timestamp":"` + t1a + `"}`,
		},
	}
	res, err := ExtractTraceFlow(context.Background(), src, "s", "trace_id", "T1", 100, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("steps: %d", len(res.Steps))
	}
	if res.Steps[0].Message != "first" || res.Steps[1].Message != "second" {
		t.Fatalf("order: %#v", res.Steps)
	}
	if res.Steps[0].Service != "api" {
		t.Fatalf("service: %q", res.Steps[0].Service)
	}
	if res.Steps[0].SpanID != "s1" {
		t.Fatalf("span: %q", res.Steps[0].SpanID)
	}
	if res.TraceID != "T1" {
		t.Fatalf("trace: %q", res.TraceID)
	}
}

func TestExtractTraceFlow_defaultField(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "s",
		lines: []string{
			`{"level":"INFO","msg":"x","trace_id":"Z"}`,
		},
	}
	res, err := ExtractTraceFlow(context.Background(), src, "s", "", "Z", 50, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Steps) != 1 {
		t.Fatalf("steps: %d", len(res.Steps))
	}
}
