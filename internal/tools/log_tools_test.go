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
