package source

import (
	"context"
	"fmt"
	"strings"
)

// MultiLogSource merges optional pipe, filesystem, Kubernetes, Docker, and/or
// Cloud Foundry sources. Stream IDs pipe://, k8s://, docker://, and cf:// are
// routed first; other IDs go to the file source when configured.
type MultiLogSource struct {
	name   string
	pipe   *PipeLogSource
	file   *FileLogSource
	k8s    *K8sLogSource
	docker *DockerLogSource
	cf     *CFLogSource
}

// NewMultiLogSource returns a combined source. At least one backend must be non-nil.
func NewMultiLogSource(name string, pipe *PipeLogSource, file *FileLogSource, k8s *K8sLogSource, docker *DockerLogSource, cf *CFLogSource) (*MultiLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: multi source name is empty")
	}
	if pipe == nil && file == nil && k8s == nil && docker == nil && cf == nil {
		return nil, fmt.Errorf("source: multi source needs at least one backend")
	}
	return &MultiLogSource{name: name, pipe: pipe, file: file, k8s: k8s, docker: docker, cf: cf}, nil
}

// Name implements [LogSource].
func (m *MultiLogSource) Name() string {
	return m.name
}

// List implements [LogSource].
func (m *MultiLogSource) List(ctx context.Context) ([]LogStream, error) {
	var out []LogStream
	if m.pipe != nil {
		a, err := m.pipe.List(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, a...)
	}
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
	if m.docker != nil {
		d, err := m.docker.List(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, d...)
	}
	if m.cf != nil {
		c, err := m.cf.List(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, c...)
	}
	return out, nil
}

// Read implements [LogSource].
func (m *MultiLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	sid := strings.TrimSpace(q.StreamID)
	if sid == PipeStreamID {
		if m.pipe == nil {
			return nil, fmt.Errorf("source: pipe log source is not configured")
		}
		return m.pipe.Read(ctx, q)
	}
	if strings.HasPrefix(q.StreamID, k8sStreamPrefix) {
		if m.k8s == nil {
			return nil, fmt.Errorf("source: kubernetes log source is not configured")
		}
		return m.k8s.Read(ctx, q)
	}
	if strings.HasPrefix(q.StreamID, dockerStreamPrefix) {
		if m.docker == nil {
			return nil, fmt.Errorf("source: docker log source is not configured")
		}
		return m.docker.Read(ctx, q)
	}
	if strings.HasPrefix(q.StreamID, cfStreamPrefix) {
		if m.cf == nil {
			return nil, fmt.Errorf("source: cloud foundry log source is not configured")
		}
		return m.cf.Read(ctx, q)
	}
	if m.file == nil {
		return nil, fmt.Errorf("source: file log source is not configured")
	}
	return m.file.Read(ctx, q)
}
