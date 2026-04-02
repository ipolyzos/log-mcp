package errsummary

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ipolyzos/log-mcp/internal/parse"
	"github.com/ipolyzos/log-mcp/internal/source"
)

func TestFingerprint_collapsesDynamicContent(t *testing.T) {
	t.Parallel()
	a := Fingerprint(`payment failed for user 1234567890123456789`)
	b := Fingerprint(`payment failed for user 9876543210987654321`)
	if a != b {
		t.Fatalf("expected same fingerprint for different long ids:\n%q\n%q", a, b)
	}
}

func TestFingerprint_distinctMessages(t *testing.T) {
	t.Parallel()
	a := Fingerprint(`timeout connecting upstream`)
	b := Fingerprint(`connection refused`)
	if a == b {
		t.Fatal("expected different fingerprints")
	}
}

func TestRawSignature_errorFieldsOrderIndependent(t *testing.T) {
	t.Parallel()
	p1 := parseLine(t, `boom`, map[string]string{"code": "500", "err": "x"})
	p2 := parseLine(t, `boom`, map[string]string{"err": "x", "code": "500"})
	r1 := rawSignature(p1, "")
	r2 := rawSignature(p2, "")
	if r1 != r2 {
		t.Fatalf("order-dependent signatures:\n%q\n%q", r1, r2)
	}
	if Fingerprint(r1) != Fingerprint(r2) {
		t.Fatal("fingerprints differ")
	}
}

func TestRawSignature_extraFieldsChangeGroup(t *testing.T) {
	t.Parallel()
	p1 := parseLine(t, `fail`, map[string]string{"code": "A"})
	p2 := parseLine(t, `fail`, map[string]string{"code": "B"})
	if Fingerprint(rawSignature(p1, "")) == Fingerprint(rawSignature(p2, "")) {
		t.Fatal("expected different groups for different codes")
	}
}

func TestServiceLabel_knownKeys(t *testing.T) {
	t.Parallel()
	got := serviceLabel(map[string]string{"service": "checkout-api"}, "app.log")
	if got != "checkout-api" {
		t.Fatalf("got %q", got)
	}
	got = serviceLabel(map[string]string{"Service_Name": "legacy"}, "x.log")
	if got != "legacy" {
		t.Fatalf("got %q", got)
	}
}

func TestServiceLabel_fallback(t *testing.T) {
	t.Parallel()
	got := serviceLabel(map[string]string{"other": "v"}, "payments.log")
	if got != "payments.log" {
		t.Fatalf("got %q", got)
	}
}

func TestIsErrorLevel(t *testing.T) {
	t.Parallel()
	for _, l := range []string{"ERROR", "error", "Fatal", "CRITICAL"} {
		if !isErrorLevel(l) {
			t.Errorf("expected error level %q", l)
		}
	}
	for _, l := range []string{"INFO", "WARN", "DEBUG", ""} {
		if isErrorLevel(l) {
			t.Errorf("not error level %q", l)
		}
	}
}

func TestSummarize_kubernetesCRI_stderrInfersError(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "pod",
		lines: []string{
			`2024-06-15T10:30:01.123456789Z stdout F GET /healthz HTTP/1.1 200`,
			`2024-06-15T10:30:02Z stderr F connection refused to redis.cache.svc.cluster.local:6379`,
		},
	}
	sum, err := Summarize(context.Background(), src, Options{StreamID: "pod", MaxScanLines: 10})
	if err != nil {
		t.Fatal(err)
	}
	if sum.ErrorLineCount != 1 {
		t.Fatalf("error lines: got %d want 1", sum.ErrorLineCount)
	}
	if sum.ScannedLines != 2 {
		t.Fatalf("scanned: got %d want 2", sum.ScannedLines)
	}
	if len(sum.Groups) != 1 {
		t.Fatalf("groups: %+v", sum.Groups)
	}
}

func TestSummarize_groupingRankingAndServiceAgg(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "stream",
		lines: []string{
			`{"level":"INFO","msg":"ok"}`,
			`{"level":"ERROR","msg":"alpha flake","service":"api"}`,
			`{"level":"ERROR","msg":"alpha flake","service":"api"}`,
			`{"level":"ERROR","msg":"beta bug","service":"worker"}`,
			`{"level":"ERROR","msg":"beta bug","service":"api"}`,
		},
	}
	sum, err := Summarize(context.Background(), src, Options{StreamID: "stream", MaxScanLines: 100})
	if err != nil {
		t.Fatal(err)
	}
	if sum.ErrorLineCount != 4 {
		t.Fatalf("error lines: got %d want 4", sum.ErrorLineCount)
	}
	if len(sum.Groups) != 2 {
		t.Fatalf("groups: got %d want 2: %+v", len(sum.Groups), sum.Groups)
	}
	// alpha flake x2 → rank 1 (count 2), beta bug x2 but fingerprint > alpha → still count 2
	// tie-break fingerprint lexicographic: "alpha..." < "beta..."
	g0, g1 := sum.Groups[0], sum.Groups[1]
	if g0.Count != 2 || g1.Count != 2 {
		t.Fatalf("counts: %+v %+v", g0, g1)
	}
	if g0.Rank != 1 || g1.Rank != 2 {
		t.Fatalf("ranks: %+v %+v", g0.Rank, g1.Rank)
	}
	if g0.Fingerprint >= g1.Fingerprint {
		t.Fatalf("expected ascending fingerprint after tie: %q >= %q", g0.Fingerprint, g1.Fingerprint)
	}
	// beta group: 1 worker + 1 api
	var beta *Group
	for i := range sum.Groups {
		if strings.Contains(strings.ToLower(sum.Groups[i].Pattern), "beta") {
			beta = &sum.Groups[i]
			break
		}
	}
	if beta == nil {
		t.Fatal("beta group not found")
	}
	if len(beta.ByService) != 2 {
		t.Fatalf("by_service: %+v", beta.ByService)
	}
}

// sliceSource is a test-only [source.LogSource] with in-memory lines.
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

func parseLine(t *testing.T, msg string, fields map[string]string) parse.ParsedLine {
	t.Helper()
	p := parse.ParsedLine{Message: msg, Fields: fields}
	return p
}
