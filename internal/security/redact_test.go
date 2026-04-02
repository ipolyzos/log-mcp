package security

import (
	"strings"
	"testing"
)

func TestRedactLine_masksToken(t *testing.T) {
	t.Parallel()
	s := `msg=ghp_abcdefghijklmnopqrstuvwxyz123456`
	out := RedactLine(s)
	if strings.Contains(out, "ghp_") {
		t.Fatalf("token leaked: %q", out)
	}
	if !strings.Contains(out, "<token>") {
		t.Fatalf("expected placeholder: %q", out)
	}
}

func TestRedactionEngine_Apply(t *testing.T) {
	t.Parallel()
	var eng RedactionEngine
	out := eng.Apply(`{"token":"sk-123456789012345678901234567890"}`)
	if strings.Contains(out, "sk-") {
		t.Fatalf("expected masking: %q", out)
	}
}
