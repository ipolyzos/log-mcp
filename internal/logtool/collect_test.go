package logtool

import (
	"context"
	"fmt"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/source"
)

type sliceSource struct {
	id    string
	name  string
	lines []string
}

func (s *sliceSource) Name() string {
	if s.name != "" {
		return s.name
	}
	return "slice"
}

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
		var limit int
		if q.MaxLines > 0 {
			limit = int(q.MaxLines)
		} else {
			limit = len(s.lines)
		}
		for i := 0; i < limit && i < len(s.lines); i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- source.LogEntry{LineNumber: int64(i + 1), Raw: s.lines[i]}:
			}
		}
	}()
	return ch, nil
}

func TestCollect_unfilteredRespectsSourceMaxLines(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "a",
		lines: []string{
			`{"level":"INFO","msg":"one"}`,
			`{"level":"INFO","msg":"two"}`,
			`{"level":"INFO","msg":"three"}`,
		},
	}
	lines, trunc, err := Collect(context.Background(), src, source.Query{StreamID: "a", MaxLines: 2}, Filter{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if trunc {
		t.Fatal("unexpected truncated")
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if lines[1].LineNumber != 2 {
		t.Fatalf("line: %+v", lines[1])
	}
}

func TestCollect_filterKeywordCaps(t *testing.T) {
	t.Parallel()
	var lines []string
	for i := range 20 {
		lines = append(lines, `{"level":"INFO","msg":"x"}`)
		if i == 5 || i == 7 {
			lines[len(lines)-1] = `{"level":"INFO","msg":"needle"}`
		}
	}
	src := &sliceSource{id: "b", lines: lines}
	out, trunc, err := Collect(context.Background(), src, source.Query{StreamID: "b"}, Filter{Keyword: "needle"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !trunc {
		t.Fatal("expected truncated after cap")
	}
	if len(out) != 1 || out[0].Message != "needle" {
		t.Fatalf("got %+v", out)
	}
}
