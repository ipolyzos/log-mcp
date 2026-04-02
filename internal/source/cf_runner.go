package source

import (
	"context"
	"os/exec"
)

// CFRunner runs the Cloud Foundry CLI with explicit argument slices only (no shell).
type CFRunner interface {
	CombinedOutput(ctx context.Context, args ...string) ([]byte, error)
}

// ExecCFRunner invokes the cf binary on PATH or [Path] if set.
type ExecCFRunner struct {
	Path string
}

func (e *ExecCFRunner) bin() string {
	if e.Path != "" {
		return e.Path
	}
	return "cf"
}

// CombinedOutput implements [CFRunner].
func (e *ExecCFRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.bin(), args...)
	return cmd.Output()
}
