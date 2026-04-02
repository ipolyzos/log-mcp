package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipolyzos/log-mcp/internal/source"
)

// seqSource streams N synthetic JSON lines without storing them all in a slice.
type seqSource struct {
	streamID string
	n        int64
}

func (s *seqSource) Name() string { return "seq" }

func (s *seqSource) List(context.Context) ([]source.LogStream, error) {
	return []source.LogStream{{ID: s.streamID, Name: "test"}}, nil
}

func (s *seqSource) Read(ctx context.Context, q source.Query) (<-chan source.LogEntry, error) {
	if q.StreamID != s.streamID {
		return nil, fmt.Errorf("bad stream")
	}
	limit := s.n
	if q.MaxLines > 0 && q.MaxLines < limit {
		limit = q.MaxLines
	}
	ch := make(chan source.LogEntry, 1)
	go func() {
		defer close(ch)
		for i := int64(1); i <= limit; i++ {
			// Include a token-like substring so normalization does real work.
			line := fmt.Sprintf(
				`{"level":"info","msg":"evt %d bearer sk-test123456789012345678901234567890"}`,
				i,
			)
			select {
			case <-ctx.Done():
				return
			case ch <- source.LogEntry{LineNumber: i, Raw: line}:
			}
		}
	}()
	return ch, nil
}

func TestPipeline_orderAndRedaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "one.log")
	// ghp_* masking requires 20+ alphanumerics after the prefix (see opaquePrefixRE).
	if err := os.WriteFile(path, []byte(`{"level":"WARN","msg":"key=ghp_abcdefghijklmnopqrstuvwxyz123456"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := source.NewFileLogSource("f", root)
	if err != nil {
		t.Fatal(err)
	}
	var levels []string
	var mu sync.Mutex
	an := analyzerFunc(func(_ context.Context, n NormalizedEntry) (Analysis, error) {
		mu.Lock()
		defer mu.Unlock()
		lvl := n.Parsed.Level
		if lvl == "" {
			lvl = "?"
		}
		levels = append(levels, lvl)
		return Analysis{Level: lvl}, nil
	})
	p := &Pipeline{
		Source:        fs,
		Query:         source.Query{StreamID: path, MaxLines: 10},
		Analyzer:      an,
		ChannelBuffer: 2,
	}
	out, err := p.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for r := range out {
		n++
		if r.LineNumber != 1 {
			t.Errorf("LineNumber = %d", r.LineNumber)
		}
		if strings.Contains(r.Redacted, "ghp_") {
			t.Errorf("expected ghp token masked in redacted: %q", r.Redacted)
		}
		if !strings.Contains(r.Redacted, "<token>") {
			t.Errorf("expected <token> placeholder in redacted line: %q", r.Redacted)
		}
	}
	if n != 1 {
		t.Errorf("got %d results, want 1", n)
	}
	if len(levels) != 1 || levels[0] != "WARN" {
		t.Errorf("levels = %v", levels)
	}
}

func TestPipeline_highVolumeStreaming(t *testing.T) {
	t.Parallel()
	total := int64(50_000)
	if testing.Short() {
		total = 5_000
	}
	ctx := context.Background()
	src := &seqSource{streamID: "x", n: total}
	var counted atomic.Int64
	var badMask atomic.Int64
	an := analyzerFunc(func(_ context.Context, n NormalizedEntry) (Analysis, error) {
		if !strings.Contains(n.Redacted, "<token>") {
			badMask.Add(1)
		}
		counted.Add(1)
		return Analysis{Level: n.Parsed.Level}, nil
	})
	p := &Pipeline{
		Source:        src,
		Query:         source.Query{StreamID: "x"},
		Analyzer:      an,
		ChannelBuffer: 4,
	}
	out, err := p.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var lastLine int64
	for r := range out {
		lastLine = r.LineNumber
	}
	if counted.Load() != total {
		t.Errorf("analyzer saw %d lines, want %d", counted.Load(), total)
	}
	if badMask.Load() != 0 {
		t.Errorf("%d lines missing <token> mask", badMask.Load())
	}
	if lastLine != total {
		t.Errorf("last line = %d, want %d", lastLine, total)
	}
}

func TestPipeline_contextCancelStopsSource(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	src := &seqSource{streamID: "x", n: 1_000_000}
	p := &Pipeline{
		Source:        src,
		Query:         source.Query{StreamID: "x"},
		ChannelBuffer: 2,
	}
	out, err := p.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := <-out; !ok {
		t.Fatal("expected at least one result before cancel")
	}
	cancel()
	deadline := time.After(5 * time.Second)
	n := 1
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout after cancel (read %d results)", n)
		case r, ok := <-out:
			if !ok {
				return
			}
			n++
			_ = r
		}
	}
}

type analyzerFunc func(context.Context, NormalizedEntry) (Analysis, error)

func (f analyzerFunc) Analyze(ctx context.Context, n NormalizedEntry) (Analysis, error) {
	return f(ctx, n)
}
