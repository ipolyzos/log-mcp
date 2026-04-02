package tools

import "testing"

func TestBuildFilter_keywordXorKeywordsAny(t *testing.T) {
	t.Parallel()
	_, err := buildFilter("", "", "", "a", []string{"b"})
	if err == nil {
		t.Fatal("expected error when both keyword and keywords_any")
	}
	f, err := buildFilter("", "", "", "", []string{"  x ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.KeywordsAny) != 1 || f.KeywordsAny[0] != "x" {
		t.Fatalf("got %#v", f.KeywordsAny)
	}
}

func TestLogsEmptyHint(t *testing.T) {
	t.Parallel()
	if g := logsEmptyHint(true, true, false, 0); g != "no_lines_in_stream" {
		t.Fatalf("got %q", g)
	}
	if g := logsEmptyHint(true, false, false, 0); g != "no_matching_lines" {
		t.Fatalf("got %q", g)
	}
	if g := logsEmptyHint(false, false, false, 0); g != "no_matching_lines" {
		t.Fatalf("got %q", g)
	}
	if g := logsEmptyHint(true, true, false, 3); g != "" {
		t.Fatalf("got %q", g)
	}
	if g := logsEmptyHint(true, true, true, 0); g != "" {
		t.Fatalf("got %q", g)
	}
}
