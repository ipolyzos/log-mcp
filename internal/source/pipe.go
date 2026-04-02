package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// PipeStreamID is the stream_id clients use for logs read from an alternate file
// descriptor (see LOG_MCP_PIPE_FD). MCP JSON-RPC stdio uses fds 0–2, so piped
// application logs must use fd >= 3 (e.g. shell exec 3<file or 3< <(cat ...)).
const PipeStreamID = "pipe://"

// Reasonable upper bound; avoids garbage env values.
const maxPipeFD = 4095

// PipeLogSource streams line-oriented text from an [io.ReadCloser] opened per
// [LogSource.Read] call (each tool read gets a fresh open — typical for one-shot pipes).
type PipeLogSource struct {
	name     string
	openRead func() (io.ReadCloser, error)
}

// NewPipeLogSourceFromFD wraps an inherited OS file descriptor (e.g. 3). The fd
// must be >= 3 so the MCP stdio transport can keep using stdin/stdout/stderr.
func NewPipeLogSourceFromFD(name string, fd int) (*PipeLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: pipe source name is empty")
	}
	if fd < 3 {
		return nil, fmt.Errorf("source: pipe fd must be >= 3 (stdio MCP uses fds 0-2)")
	}
	if fd > maxPipeFD {
		return nil, fmt.Errorf("source: pipe fd %d exceeds max %d", fd, maxPipeFD)
	}
	return &PipeLogSource{
		name: name,
		openRead: func() (io.ReadCloser, error) {
			// #nosec G115 -- fd validated above to [3, maxPipeFD]; OS handle fits uintptr on supported platforms
			f := os.NewFile(uintptr(fd), "pipe")
			if f == nil {
				return nil, fmt.Errorf("source: invalid pipe fd %d", fd)
			}
			return f, nil
		},
	}, nil
}

// NewPipeLogSourceOpener is for tests and advanced wiring; open must return a
// new reader for each call.
func NewPipeLogSourceOpener(name string, open func() (io.ReadCloser, error)) (*PipeLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: pipe source name is empty")
	}
	if open == nil {
		return nil, fmt.Errorf("source: pipe open func is nil")
	}
	return &PipeLogSource{name: name, openRead: open}, nil
}

// Name implements [LogSource].
func (p *PipeLogSource) Name() string {
	return p.name
}

// List implements [LogSource].
func (p *PipeLogSource) List(ctx context.Context) ([]LogStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []LogStream{{ID: PipeStreamID, Name: "piped logs"}}, nil
}

// Read implements [LogSource].
func (p *PipeLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	if q.StreamID == "" {
		return nil, fmt.Errorf("source: empty stream id")
	}
	if strings.TrimSpace(q.StreamID) != PipeStreamID {
		return nil, fmt.Errorf("source: stream %q is not the pipe source (use %q)", q.StreamID, PipeStreamID)
	}
	rc, err := p.openRead()
	if err != nil {
		return nil, err
	}
	ch := make(chan LogEntry, 64)
	go streamReaderLines(ctx, rc, q, ch)
	return ch, nil
}
