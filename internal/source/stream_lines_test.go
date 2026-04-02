package source

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestStreamReaderLines_truncatesByMaxLineBytes(t *testing.T) {
	t.Parallel()
	r := io.NopCloser(strings.NewReader("abcdefghij\n"))
	ch := make(chan LogEntry, 4)
	q := Query{MaxLineBytes: 4}
	go streamReaderLines(context.Background(), r, q, ch)
	var got LogEntry
	for e := range ch {
		got = e
	}
	if got.Raw != "abcd" {
		t.Fatalf("got %q want abcd", got.Raw)
	}
}
