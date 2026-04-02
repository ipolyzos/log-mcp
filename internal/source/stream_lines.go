package source

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// streamReaderLines reads line-oriented text from r into ch until EOF, context
// cancellation, or Query limits. It closes ch and r when done.
func streamReaderLines(ctx context.Context, r io.ReadCloser, q Query, ch chan<- LogEntry) {
	defer close(ch)
	defer func() { _ = r.Close() }()

	br := bufio.NewReaderSize(r, 1<<20)
	var lineNum, bytesRead int64
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		line, err := br.ReadString('\n')
		if line != "" {
			lineNum++
			raw := strings.TrimSuffix(strings.TrimSuffix(line, "\r\n"), "\n")
			raw = truncateLineBytes(raw, q.MaxLineBytes)
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

func truncateLineBytes(raw string, max int64) string {
	if max <= 0 || int64(len(raw)) <= max {
		return raw
	}
	return raw[:max]
}
