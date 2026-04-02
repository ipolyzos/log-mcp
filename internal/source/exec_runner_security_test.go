package source

import (
	"context"
	"testing"
)

func TestExecKubectlRunner_validateBeforeExec(t *testing.T) {
	t.Parallel()
	r := &ExecKubectlRunner{Path: "kubectl"}
	_, err := r.CombinedOutput(context.Background(), "exec", "-it", "pod", "--", "sh")
	if err == nil {
		t.Fatal("expected validation error before exec")
	}
}

func TestExecDockerRunner_validateBeforeExec(t *testing.T) {
	t.Parallel()
	r := &ExecDockerRunner{Path: "docker"}
	_, err := r.CombinedOutput(context.Background(), "run", "-it", "alpine", "sh")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestExecCFRunner_validateBeforeExec(t *testing.T) {
	t.Parallel()
	r := &ExecCFRunner{Path: "cf"}
	_, err := r.CombinedOutput(context.Background(), "ssh", "myapp")
	if err == nil {
		t.Fatal("expected validation error")
	}
}
