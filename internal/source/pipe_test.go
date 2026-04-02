package source

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipeLogSource_Read_opener(t *testing.T) {
	t.Parallel()
	body := "line one\nline two\n"
	p, err := NewPipeLogSourceOpener("p", func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].ID != PipeStreamID {
		t.Fatalf("List: %+v", streams)
	}
	ch, err := p.Read(ctx, Query{StreamID: PipeStreamID})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for e := range ch {
		lines = append(lines, e.Raw)
	}
	if len(lines) != 2 || lines[0] != "line one" || lines[1] != "line two" {
		t.Fatalf("lines: %#v", lines)
	}
}

func TestPipeLogSource_Read_wrongStreamID(t *testing.T) {
	t.Parallel()
	p, err := NewPipeLogSourceOpener("p", func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("x\n")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Read(context.Background(), Query{StreamID: "/tmp/nope"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPipeLogSourceFromFD_rejectsStdFDs(t *testing.T) {
	t.Parallel()
	for _, fd := range []int{-1, 0, 1, 2} {
		if _, err := NewPipeLogSourceFromFD("p", fd); err == nil {
			t.Fatalf("fd %d: expected error", fd)
		}
	}
}

func TestMultiLogSource_routesPipe(t *testing.T) {
	t.Parallel()
	p, err := NewPipeLogSourceOpener("pipe", func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("piped\n")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "f.log")
	if err := os.WriteFile(logPath, []byte("disk\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileSrc, err := NewFileLogSource("files", dir)
	if err != nil {
		t.Fatal(err)
	}
	multi, err := NewMultiLogSource("pf", p, fileSrc, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := multi.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Fatalf("want 2 streams, got %d: %+v", len(streams), streams)
	}
	ch, err := multi.Read(ctx, Query{StreamID: PipeStreamID})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for range ch {
		n++
	}
	if n != 1 {
		t.Fatalf("pipe lines %d", n)
	}
}
