package llmtools

import "testing"

func TestExplainErrorGroup(t *testing.T) {
	t.Parallel()
	exp, err := ExplainErrorGroup(ErrorGroupPattern{Rank: 1, Count: 10, Pattern: "timeout", Fingerprint: "fp"})
	if err != nil {
		t.Fatal(err)
	}
	if exp.PatternKind != "summarize_errors.group" {
		t.Fatalf("kind: %s", exp.PatternKind)
	}
}

func TestExplainRepeatedBurst(t *testing.T) {
	t.Parallel()
	exp, err := ExplainRepeatedBurst(RepeatedBurstPattern{Count: 8, FirstLine: 1, LastLine: 20, SampleMessage: "err"})
	if err != nil {
		t.Fatal(err)
	}
	if exp.PatternKind != "detect_anomalies.repeated_errors" {
		t.Fatalf("kind: %s", exp.PatternKind)
	}
}

func TestExplainErrorGroup_rejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := ExplainErrorGroup(ErrorGroupPattern{})
	if err == nil {
		t.Fatal("expected error")
	}
}
