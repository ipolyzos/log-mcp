package logtool

import (
	"testing"
	"time"

	"github.com/ipolyzos/log-mcp/internal/parse"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
)

func TestFilter_Empty(t *testing.T) {
	t.Parallel()
	if !(Filter{}).Empty() {
		t.Fatal("expected empty")
	}
	f := Filter{Level: "INFO"}
	if f.Empty() {
		t.Fatal("expected non-empty")
	}
}

func TestMatch_keywordLevelTime(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	base := pipeline.Result{
		LineNumber: 1,
		Redacted:   `{"msg":"hello world"}`,
		Parsed: parse.ParsedLine{
			Timestamp: &ts,
			Level:     "INFO",
			Message:   "hello world",
		},
	}
	if !Match(Filter{Keyword: "hello"}, base) {
		t.Fatal("keyword should match")
	}
	if Match(Filter{Keyword: "nope"}, base) {
		t.Fatal("keyword should not match")
	}
	if !Match(Filter{Level: "info"}, base) {
		t.Fatal("level equal fold")
	}
	if Match(Filter{Level: "WARN"}, base) {
		t.Fatal("level mismatch")
	}
	since := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !Match(Filter{Since: &since, Until: &until}, base) {
		t.Fatal("time in range")
	}
	before := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if Match(Filter{Since: &before}, base) {
		t.Fatal("since after line")
	}
	noTS := base
	noTS.Parsed.Timestamp = nil
	if Match(Filter{Since: &since}, noTS) {
		t.Fatal("missing timestamp should not match time filter")
	}
}
