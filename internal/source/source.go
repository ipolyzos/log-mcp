package source

import "context"

// LogStream identifies one readable log stream exposed by a [LogSource]
// (for example a single log file).
type LogStream struct {
	ID   string
	Name string
}

// Query selects what to read from a source. StreamID must match a stream
// returned by [LogSource.List] for that source (for file sources this is
// typically an absolute path).
type Query struct {
	StreamID string
	// MaxLines stops after this many lines; 0 means no limit.
	MaxLines int64
	// MaxBytes stops after approximately this many raw bytes of line payloads
	// (excluding internal framing); 0 means no limit.
	MaxBytes int64
	// MaxLineBytes caps each raw line length in bytes; 0 means no per-line cap.
	MaxLineBytes int64
}

// LogEntry is one logical line from a log stream.
type LogEntry struct {
	LineNumber int64
	Raw        string
}

// LogSource is a read-only backend that can enumerate streams and read them
// as a line-oriented stream.
type LogSource interface {
	Name() string
	List(ctx context.Context) ([]LogStream, error)
	Read(ctx context.Context, query Query) (<-chan LogEntry, error)
}
