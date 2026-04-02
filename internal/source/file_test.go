package source

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFileLogSource_ListAndRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	content := "alpha\nbeta\ngamma\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := NewFileLogSource("files", dir)
	if err != nil {
		t.Fatal(err)
	}
	if g, w := src.Name(), "files"; g != w {
		t.Fatalf("Name: got %q want %q", g, w)
	}

	ctx := context.Background()
	streams, err := src.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 {
		t.Fatalf("List: got %d streams want 1", len(streams))
	}
	if streams[0].Name != "app.log" {
		t.Fatalf("stream name: got %q", streams[0].Name)
	}

	ch, err := src.Read(ctx, Query{StreamID: streams[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for e := range ch {
		lines = append(lines, e.Raw)
	}
	want := []string{"alpha", "beta", "gamma"}
	if len(lines) != len(want) {
		t.Fatalf("lines: got %v want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, lines[i], want[i])
		}
	}
}

func TestFileLogSource_ReadMaxLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "x.log")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := NewFileLogSource("f", path)
	if err != nil {
		t.Fatal(err)
	}
	streams, err := src.List(context.Background())
	if err != nil || len(streams) != 1 {
		t.Fatalf("List: %v streams=%d", err, len(streams))
	}
	ch, err := src.Read(context.Background(), Query{StreamID: streams[0].ID, MaxLines: 2})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for range ch {
		n++
	}
	if n != 2 {
		t.Fatalf("got %d lines want 2", n)
	}
}

func TestFileLogSource_LargeLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "wide.log")
	// Much larger than bufio.Scanner's default max token size (64 KiB).
	const width = 256 * 1024
	longLine := strings.Repeat("x", width) + "\n"
	if err := os.WriteFile(path, []byte(longLine+"tail\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := NewFileLogSource("wide", path)
	if err != nil {
		t.Fatal(err)
	}
	streams, err := src.List(context.Background())
	if err != nil || len(streams) != 1 {
		t.Fatalf("List: %v", err)
	}
	ch, err := src.Read(context.Background(), Query{StreamID: streams[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	var first LogEntry
	var count int
	for e := range ch {
		if count == 0 {
			first = e
		}
		count++
	}
	if count != 2 {
		t.Fatalf("line count: got %d want 2", count)
	}
	if len(first.Raw) != width {
		t.Fatalf("first line len: got %d want %d", len(first.Raw), width)
	}
	if first.LineNumber != 1 {
		t.Fatalf("LineNumber: got %d", first.LineNumber)
	}
}

func TestFileLogSource_LargeLogManyLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "many.log")
	const nLines = 50_000
	var b strings.Builder
	b.Grow(nLines * 8)
	for i := range nLines {
		b.WriteString("line-")
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := NewFileLogSource("many", path)
	if err != nil {
		t.Fatal(err)
	}
	streams, err := src.List(context.Background())
	if err != nil || len(streams) != 1 {
		t.Fatal(err)
	}
	ch, err := src.Read(context.Background(), Query{StreamID: streams[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	var got int
	var last LogEntry
	for e := range ch {
		got++
		last = e
	}
	if got != nLines {
		t.Fatalf("lines read: got %d want %d", got, nLines)
	}
	if last.LineNumber != nLines {
		t.Fatalf("last LineNumber: got %d", last.LineNumber)
	}
	if want := "line-" + strconv.Itoa(nLines-1); last.Raw != want {
		t.Fatalf("last line: got %q want %q", last.Raw, want)
	}
}

func TestFileLogSource_ReadRejectsPathOutsideRoot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	other := t.TempDir()
	evil := filepath.Join(other, "secret.log")
	if err := os.WriteFile(evil, []byte("no\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := NewFileLogSource("sandbox", dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Read(context.Background(), Query{StreamID: evil})
	if err == nil {
		t.Fatal("expected error for path outside root")
	}
}

func TestFileLogSource_Interface(t *testing.T) {
	var _ LogSource = (*FileLogSource)(nil)
}
