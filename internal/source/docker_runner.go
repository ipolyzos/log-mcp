package source

import (
	"context"
	"io"
	"os/exec"
	"sync"
)

// DockerRunner runs docker with explicit argument slices only (no shell).
type DockerRunner interface {
	CombinedOutput(ctx context.Context, args ...string) ([]byte, error)
	StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error)
}

// ExecDockerRunner invokes the docker binary on PATH or [Path] if set.
type ExecDockerRunner struct {
	Path string
}

func (e *ExecDockerRunner) bin() string {
	if e.Path != "" {
		return e.Path
	}
	return "docker"
}

// CombinedOutput implements [DockerRunner].
func (e *ExecDockerRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.bin(), args...)
	return cmd.Output()
}

// StdoutPipe implements [DockerRunner].
func (e *ExecDockerRunner) StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, e.bin(), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, _ := cmd.StderrPipe()
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &dockerWaitReader{r: stdout, cmd: cmd}, nil
}

type dockerWaitReader struct {
	r    io.ReadCloser
	cmd  *exec.Cmd
	mu   sync.Mutex
	err  error
	done bool
}

func (d *dockerWaitReader) Read(p []byte) (int, error) {
	return d.r.Read(p)
}

func (d *dockerWaitReader) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.done {
		return d.err
	}
	d.done = true
	_ = d.r.Close()
	d.err = d.cmd.Wait()
	return d.err
}
