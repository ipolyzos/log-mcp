package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileLogSource exposes regular files under configured roots as log streams.
// A root may be a file (one stream) or a directory (one stream per immediate
// child regular file; subdirectories are not walked).
type FileLogSource struct {
	name string
	// rootsAbs are clean absolute paths (files or directories).
	rootsAbs []string
}

// NewFileLogSource validates roots, resolves them to absolute paths, and
// returns a source named name. At least one root is required.
func NewFileLogSource(name string, roots ...string) (*FileLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: file source name is empty")
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("source: file source needs at least one root")
	}
	absRoots := make([]string, 0, len(roots))
	for _, r := range roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			return nil, fmt.Errorf("source: resolve root %q: %w", r, err)
		}
		abs = filepath.Clean(abs)
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("source: stat root %q: %w", abs, err)
		}
		absRoots = append(absRoots, abs)
	}
	return &FileLogSource{name: name, rootsAbs: absRoots}, nil
}

// Name implements [LogSource].
func (f *FileLogSource) Name() string {
	return f.name
}

// List implements [LogSource].
func (f *FileLogSource) List(ctx context.Context) ([]LogStream, error) {
	var out []LogStream
	for _, root := range f.rootsAbs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fi, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			out = append(out, LogStream{ID: root, Name: filepath.Base(root)})
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			path := filepath.Join(root, e.Name())
			out = append(out, LogStream{ID: path, Name: e.Name()})
		}
	}
	return out, nil
}

// Read implements [LogSource]. The returned channel is closed when the read
// completes, the context is cancelled, MaxLines/MaxBytes is reached, or an I/O
// error occurs. The caller must consume until the channel is closed.
func (f *FileLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	if q.StreamID == "" {
		return nil, fmt.Errorf("source: empty stream id")
	}
	path, err := filepath.Abs(q.StreamID)
	if err != nil {
		return nil, err
	}
	path = filepath.Clean(path)
	if !f.allowedFile(path) {
		return nil, fmt.Errorf("source: stream %q is not allowed for this file source", q.StreamID)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	ch := make(chan LogEntry, 64)
	go f.streamFile(ctx, file, q, ch)
	return ch, nil
}

func (f *FileLogSource) allowedFile(abs string) bool {
	fi, err := os.Stat(abs)
	if err != nil || fi.IsDir() || !fi.Mode().IsRegular() {
		return false
	}
	for _, root := range f.rootsAbs {
		if rel, err := filepath.Rel(root, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (f *FileLogSource) streamFile(ctx context.Context, file *os.File, q Query, ch chan<- LogEntry) {
	defer close(ch)
	defer func() { _ = file.Close() }()

	br := bufio.NewReaderSize(file, 1<<20) // 1 MiB buffer for wide lines / throughput
	var (
		lineNum   int64
		bytesRead int64
	)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		line, err := br.ReadString('\n')
		if line != "" {
			lineNum++
			raw := strings.TrimSuffix(strings.TrimSuffix(line, "\r\n"), "\n")
			if q.MaxLines > 0 && lineNum > q.MaxLines {
				return
			}
			n := int64(len(raw))
			if q.MaxBytes > 0 && bytesRead+n > q.MaxBytes {
				return
			}
			bytesRead += n
			select {
			case <-ctx.Done():
				return
			case ch <- LogEntry{LineNumber: lineNum, Raw: raw}:
			}
		}
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
	}
}
