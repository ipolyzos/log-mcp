package security

import (
	"strings"
	"testing"
)

func TestValidateKubectlArgs_getPodsAllowlisted(t *testing.T) {
	t.Parallel()
	args := []string{"get", "pods", "-n", "default", "-o", "jsonpath=" + k8sPodListJSONPath}
	if err := ValidateKubectlArgs(args); err != nil {
		t.Fatal(err)
	}
}

func TestValidateKubectlArgs_getPodsRejectsWrongJSONPath(t *testing.T) {
	t.Parallel()
	args := []string{"get", "pods", "-n", "default", "-o", "jsonpath={.items[*]}"}
	if err := ValidateKubectlArgs(args); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateKubectlArgs_logsAllowlisted(t *testing.T) {
	t.Parallel()
	cases := [][]string{
		{"logs", "my-pod", "-n", "prod"},
		{"logs", "my-pod", "-n", "prod", "-c", "sidecar"},
		{"logs", "my-pod", "-n", "prod", "--tail", "100"},
		{"logs", "my-pod", "-n", "prod", "-c", "c1", "--tail", "10"},
	}
	for _, args := range cases {
		if err := ValidateKubectlArgs(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

func TestValidateKubectlArgs_rejectsInjection(t *testing.T) {
	t.Parallel()
	args := []string{"logs", "pod", "-n", "default\n--evil"}
	if err := ValidateKubectlArgs(args); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateKubectlArgs_rejectsUnknownSubcommand(t *testing.T) {
	t.Parallel()
	if err := ValidateKubectlArgs([]string{"delete", "pod", "x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDockerArgs(t *testing.T) {
	t.Parallel()
	if err := ValidateDockerArgs([]string{"ps", "-a", "--format", "{{.Names}}"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDockerArgs([]string{"logs", "c1"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDockerArgs([]string{"logs", "c1", "--tail", "50"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDockerArgs([]string{"logs", "c1", "--follow"}); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateDockerArgs([]string{"images"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateCFArgs(t *testing.T) {
	t.Parallel()
	if err := ValidateCFArgs([]string{"apps"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCFArgs([]string{"logs", "my-app", "--recent"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCFArgs([]string{"logs", "MyApp", "--recent"}); err == nil {
		t.Fatal("expected error for uppercase")
	}
	if err := ValidateCFArgs([]string{"push", "x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestArgClean_rejectsControlChars(t *testing.T) {
	t.Parallel()
	if err := argClean([]string{"ok", "bad\n"}); err == nil {
		t.Fatal("expected error")
	}
	if err := argClean([]string{strings.Repeat("a", 100)}); err != nil {
		t.Fatal(err)
	}
}
