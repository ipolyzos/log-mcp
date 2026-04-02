package loganalysis

import (
	"context"
	"fmt"
	"strings"
	"testing"

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
		if q.MaxLines > 0 && int(q.MaxLines) < limit {
			limit = int(q.MaxLines)
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

func TestDetectAnomalies_repeatedErrors(t *testing.T) {
	t.Parallel()
	var lines []string
	for range 10 {
		lines = append(lines, `{"level":"ERROR","msg":"same failure","code":"E1"}`)
	}
	for range 20 {
		lines = append(lines, `{"level":"INFO","msg":"ok"}`)
	}
	src := &sliceSource{id: "s", lines: lines}
	rep, err := DetectAnomalies(context.Background(), src, "s", AnomalyOptions{
		MaxScanLines:    500,
		SegmentCount:    10,
		ZScoreThreshold: 3,
		MinRepeatErrors: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ScannedLines != 30 {
		t.Fatalf("scanned: %d", rep.ScannedLines)
	}
	if len(rep.RepeatedErrors) < 1 {
		t.Fatalf("expected repeated error burst: %+v", rep.RepeatedErrors)
	}
	if rep.RepeatedErrors[0].Count < 6 {
		t.Fatalf("count: %d", rep.RepeatedErrors[0].Count)
	}
	if !rep.LowSample {
		t.Fatal("expected low_sample: all lines fall in one segment vs default width")
	}
	if len(rep.LevelSpikeSegments) != 0 {
		t.Fatalf("expected no level spikes when low_sample: %+v", rep.LevelSpikeSegments)
	}
}

func TestDetectAnomalies_levelSpike(t *testing.T) {
	t.Parallel()
	// Many INFO, then a block of ERRORs in one segment (width=10, segN=10, maxScan=100).
	var b strings.Builder
	for range 70 {
		b.WriteString(`{"level":"INFO","msg":"x"}` + "\n")
	}
	for range 25 {
		b.WriteString(`{"level":"ERROR","msg":"boom"}` + "\n")
	}
	for range 5 {
		b.WriteString(`{"level":"INFO","msg":"tail"}` + "\n")
	}
	raw := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	src := &sliceSource{id: "s", lines: raw}
	rep, err := DetectAnomalies(context.Background(), src, "s", AnomalyOptions{
		MaxScanLines:    100,
		SegmentCount:    10,
		ZScoreThreshold: 1.5,
		MinRepeatErrors: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.LevelSpikeSegments) < 1 {
		t.Fatalf("expected level spike: spikes=%v repeats=%v", rep.LevelSpikeSegments, rep.RepeatedErrors)
	}
	if rep.LowSample {
		t.Fatalf("did not expect low_sample for multi-segment stream: %+v", rep)
	}
}

func TestDetectAnomalies_kubernetesCRI_shapedLines(t *testing.T) {
	t.Parallel()
	src := &sliceSource{
		id: "k",
		lines: []string{
			`2024-06-15T10:30:01.123456789Z stdout F GET /healthz HTTP/1.1 200`,
			`2024-06-15T10:30:02Z stderr F connection refused to redis.cache.svc.cluster.local:6379`,
			`2024-06-15T10:30:03Z stdout F GET /upstream HTTP/1.1 503`,
		},
	}
	rep, err := DetectAnomalies(context.Background(), src, "k", AnomalyOptions{
		MaxScanLines:    50,
		SegmentCount:    15,
		ZScoreThreshold: 1,
		MinRepeatErrors: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ScannedLines != 3 {
		t.Fatalf("scanned: %d", rep.ScannedLines)
	}
	if !rep.LowSample {
		t.Fatalf("expected low_sample for tiny CRI stream: %+v", rep)
	}
	if len(rep.LevelSpikeSegments) != 0 {
		t.Fatalf("expected no spikes: %+v", rep.LevelSpikeSegments)
	}
}

func TestDetectAnomalies_lowSample_scannedLessThanSegmentCount(t *testing.T) {
	t.Parallel()
	var lines []string
	for range 5 {
		lines = append(lines, `{"level":"INFO","msg":"ok"}`)
	}
	src := &sliceSource{id: "s", lines: lines}
	rep, err := DetectAnomalies(context.Background(), src, "s", AnomalyOptions{
		MaxScanLines:    500,
		SegmentCount:    15,
		ZScoreThreshold: 1,
		MinRepeatErrors: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ScannedLines != 5 {
		t.Fatalf("scanned: %d", rep.ScannedLines)
	}
	if !rep.LowSample {
		t.Fatal("expected low_sample when scanned_lines < segment_count")
	}
	if len(rep.LevelSpikeSegments) != 0 {
		t.Fatalf("expected no spikes: %+v", rep.LevelSpikeSegments)
	}
}
