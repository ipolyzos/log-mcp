package source

import (
	"context"
	"os/exec"

	"github.com/ipolyzos/log-mcp/internal/security"
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
	if err := security.ValidateCFArgs(args); err != nil {
		return nil, err
	}
	// #nosec G204 -- argv allowlisted; binary path fixed; see security.HardenExecCmd
	cmd := exec.CommandContext(ctx, e.bin(), args...)
	security.HardenExecCmd(cmd)
	return cmd.Output()
}
