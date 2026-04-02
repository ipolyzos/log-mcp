package logcorr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ipolyzos/log-mcp/internal/parse"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/source"
)

type sliceSource struct {
	id    string
	lines []string
}

func (s *sliceSource) Name() string { return "slice" }

func (s *sliceSource) List(context.Context) ([]source.LogStream, error) {
	return []source.LogStream{{ID: s.id, Name: "t"}}, nil
}

func (s *sliceSource) Read(ctx context.Context, q source.Query) (<-chan source.LogEntry, error) {
	if q.StreamID != s.id {
		return nil, fmt.Errorf("bad stream")
	}
	ch := make(chan source.LogEntry, 1)
	go func() {
		defer close(ch)
		limit := len(s.lines)
		if q.MaxLines > 0 {
			limit = minInt(len(s.lines), int(q.MaxLines))
		}
		for i := 0; i < limit; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- source.LogEntry{LineNumber: int64(i + 1), Raw: s.lines[i]}:
			}
		}
	}()
	return ch, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestGroupByField_countsAndOrder(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "s",
		lines: []string{
			`{"level":"INFO","msg":"a","trace_id":"t1"}`,
			`{"level":"INFO","msg":"b","trace_id":"t2"}`,
			`{"level":"INFO","msg":"c","trace_id":"t1"}`,
			`{"level":"INFO","msg":"d","trace_id":"t1"}`,
			`{"level":"INFO","msg":"e"}`,
		},
	}
	res, err := GroupByField(context.Background(), src, "s", "trace_id", 100, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("groups: %+v", res.Groups)
	}
	if res.Groups[0].Value != "t1" || res.Groups[0].Count != 3 {
		t.Fatalf("first group: %+v", res.Groups[0])
	}
	if res.Groups[1].Value != "t2" || res.Groups[1].Count != 1 {
		t.Fatalf("second group: %+v", res.Groups[1])
	}
}

func TestSortEvents_byTimestamp(t *testing.T) {
	t.Parallel()
	tLate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	tEarly := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := []LineEvent{
		{Result: pipeline.Result{LineNumber: 2, Parsed: parse.ParsedLine{Timestamp: &tLate}}},
		{Result: pipeline.Result{LineNumber: 1, Parsed: parse.ParsedLine{Timestamp: &tEarly}}},
	}
	SortEvents(ev)
	if ev[0].Result.LineNumber != 1 {
		t.Fatalf("order: %+v", ev)
	}
}

func TestCorrelateEvents_timeline(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "s",
		lines: []string{
			`{"level":"INFO","msg":"start","trace_id":"tx","@timestamp":"2026-01-10T10:00:01Z"}`,
			`{"level":"WARN","msg":"mid","trace_id":"tx","@timestamp":"2026-01-10T10:00:00Z"}`,
			`{"level":"INFO","msg":"other","trace_id":"ty"}`,
		},
	}
	res, err := CorrelateEvents(context.Background(), src, "s", "trace_id", "tx", 50, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Events) != 2 {
		t.Fatalf("events: %d", len(res.Events))
	}
	// Sorted by time: mid (10:00:00) before start (10:00:01)
	if !containsMsg(res.Events[0].Result, "mid") || !containsMsg(res.Events[1].Result, "start") {
		t.Fatalf("order: %+v / %+v", res.Events[0].Result.Parsed.Message, res.Events[1].Result.Parsed.Message)
	}
}

func TestCorrelateEvents_requestId(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "s",
		lines: []string{
			`{"level":"INFO","msg":"a","requestId":"req-1"}`,
			`{"level":"INFO","msg":"b","requestId":"req-1"}`,
		},
	}
	res, err := CorrelateEvents(context.Background(), src, "s", "request_id", "req-1", 50, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Events) != 2 {
		t.Fatalf("events: %d", len(res.Events))
	}
}

func containsMsg(r pipeline.Result, sub string) bool {
	return len(sub) == 0 || (len(r.Parsed.Message) >= len(sub) && indexFold(r.Parsed.Message, sub) >= 0)
}

func indexFold(s, sub string) int {
	// ASCII-only test helper
	ls, lsub := toLowerASCII(s), toLowerASCII(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return i
		}
	}
	return -1
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
