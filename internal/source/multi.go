package source

import (
	"context"
	"fmt"
	"strings"
)

// MultiLogSource merges a filesystem source and/or a Kubernetes source.
// Stream IDs with prefix "k8s://" are routed to the k8s source; all others to the file source.
type MultiLogSource struct {
	name string
	file *FileLogSource
	k8s  *K8sLogSource
}

// NewMultiLogSource returns a combined source. At least one of file or k8s must be non-nil.
func NewMultiLogSource(name string, file *FileLogSource, k8s *K8sLogSource) (*MultiLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: multi source name is empty")
	}
	if file == nil && k8s == nil {
		return nil, fmt.Errorf("source: multi source needs file and/or k8s")
	}
	return &MultiLogSource{name: name, file: file, k8s: k8s}, nil
}

// Name implements [LogSource].
func (m *MultiLogSource) Name() string {
	return m.name
}

// List implements [LogSource].
func (m *MultiLogSource) List(ctx context.Context) ([]LogStream, error) {
	var out []LogStream
	if m.file != nil {
		a, err := m.file.List(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, a...)
	}
	if m.k8s != nil {
		b, err := m.k8s.List(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

// Read implements [LogSource].
func (m *MultiLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	if strings.HasPrefix(q.StreamID, k8sStreamPrefix) {
		if m.k8s == nil {
			return nil, fmt.Errorf("source: kubernetes log source is not configured")
		}
		return m.k8s.Read(ctx, q)
	}
	if m.file == nil {
		return nil, fmt.Errorf("source: file log source is not configured")
	}
	return m.file.Read(ctx, q)
}
