package source

import (
	"context"
	"io"
	"os/exec"
	"sync"
)

// KubectlRunner runs kubectl with explicit argument slices only (no shell).
// Implementations must not prepend arbitrary user-controlled tokens.
type KubectlRunner interface {
	// CombinedOutput runs kubectl once and returns stdout (stderr on error).
	CombinedOutput(ctx context.Context, args ...string) ([]byte, error)
	// StdoutPipe starts kubectl and returns stdout; Close waits for the process.
	StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error)
}

// ExecKubectlRunner invokes the kubectl binary on PATH or [Path] if set.
type ExecKubectlRunner struct {
	Path string
}

func (e *ExecKubectlRunner) bin() string {
	if e.Path != "" {
		return e.Path
	}
	return "kubectl"
}

// CombinedOutput implements [KubectlRunner].
func (e *ExecKubectlRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.bin(), args...)
	return cmd.Output()
}

// StdoutPipe implements [KubectlRunner].
func (e *ExecKubectlRunner) StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error) {
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
	return &kubectlWaitReader{r: stdout, cmd: cmd}, nil
}

type kubectlWaitReader struct {
	r    io.ReadCloser
	cmd  *exec.Cmd
	mu   sync.Mutex
	err  error
	done bool
}

func (k *kubectlWaitReader) Read(p []byte) (int, error) {
	return k.r.Read(p)
}

func (k *kubectlWaitReader) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.done {
		return k.err
	}
	k.done = true
	_ = k.r.Close()
	k.err = k.cmd.Wait()
	return k.err
}
