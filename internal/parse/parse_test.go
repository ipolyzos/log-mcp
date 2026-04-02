package parse

import (
	"strings"
	"testing"
	"time"
)

func TestJSONParser_full(t *testing.T) {
	t.Parallel()
	line := `{"@timestamp":"2024-06-01T12:00:00Z","level":"info","msg":"hello","trace_id":"abc","n":42}`
	var p JSONParser
	got, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != "INFO" {
		t.Fatalf("level: got %q", got.Level)
	}
	if got.Message != "hello" {
		t.Fatalf("message: got %q", got.Message)
	}
	if got.Timestamp == nil || !got.Timestamp.Equal(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("timestamp: %v", got.Timestamp)
	}
	if got.Fields == nil || got.Fields["trace_id"] != "abc" {
		t.Fatalf("fields trace_id: %v", got.Fields)
	}
	if got.Fields["n"] != "42" {
		t.Fatalf("fields n: got %q", got.Fields["n"])
	}
}

func TestJSONParser_unixMilliseconds(t *testing.T) {
	t.Parallel()
	const ms = int64(1_715_000_000_123)
	line := `{"ts":1715000000123,"msg":"tick"}`
	var p JSONParser
	got, err := p.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.Message != "tick" {
		t.Fatalf("message: %q", got.Message)
	}
	if got.Timestamp == nil || got.Timestamp.UnixMilli() != ms {
		t.Fatalf("timestamp ms: got %v want %d", got.Timestamp, ms)
	}
}

func TestJSONParser_malformed(t *testing.T) {
	t.Parallel()
	var p JSONParser
	if _, err := p.Parse(`{"msg":`); err == nil {
		t.Fatal("expected error for truncated json")
	}
}

func TestJSONParser_notObject(t *testing.T) {
	t.Parallel()
	var p JSONParser
	if _, err := p.Parse(`[1,2,3]`); err == nil {
		t.Fatal("expected error for json array")
	}
	if _, err := p.Parse(`"just a string"`); err == nil {
		t.Fatal("expected error for json string")
	}
}

func TestTextParser_timestampLevelMessage(t *testing.T) {
	t.Parallel()
	p := TextParser{}
	got, err := p.Parse(`2024-06-01T12:00:00Z INFO hello world`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != "INFO" {
		t.Fatalf("level: %q", got.Level)
	}
	if got.Message != "hello world" {
		t.Fatalf("message: %q", got.Message)
	}
	if got.Timestamp == nil {
		t.Fatal("expected timestamp")
	}
}

func TestTextParser_bracketAndKV(t *testing.T) {
	t.Parallel()
	p := TextParser{}
	got, err := p.Parse(`[WARN] timeout host="upstream" code=504`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != "WARN" {
		t.Fatalf("level: %q", got.Level)
	}
	if got.Fields == nil || got.Fields["host"] != "upstream" || got.Fields["code"] != "504" {
		t.Fatalf("fields: %v", got.Fields)
	}
	if !strings.Contains(got.Message, "timeout") {
		t.Fatalf("message: %q", got.Message)
	}
}

func TestTextParser_plainLine(t *testing.T) {
	t.Parallel()
	p := TextParser{}
	got, err := p.Parse(`   no special structure here   `)
	if err != nil {
		t.Fatal(err)
	}
	if got.Message != "no special structure here" {
		t.Fatalf("message: %q", got.Message)
	}
}

func TestAutoParser_mixedFormats(t *testing.T) {
	t.Parallel()
	a := AutoParser{}
	cases := []struct {
		line            string
		wantLevel       string
		wantSubstr      string
		wantJSONMessage string
	}{
		{line: `{"msg":"json-line","level":"error"}`, wantJSONMessage: "json-line", wantLevel: "ERROR"},
		{line: `2024-06-01T15:00:00Z DEBUG plain tail`, wantLevel: "DEBUG", wantSubstr: "plain tail"},
		{line: `[INFO] simple`, wantLevel: "INFO", wantSubstr: "simple"},
	}
	for _, tc := range cases {
		got, err := a.Parse(tc.line)
		if err != nil {
			t.Fatalf("line %q: %v", tc.line, err)
		}
		if tc.wantJSONMessage != "" {
			if got.Message != tc.wantJSONMessage {
				t.Fatalf("json msg: got %q want %q", got.Message, tc.wantJSONMessage)
			}
		}
		if got.Level != tc.wantLevel {
			t.Fatalf("line %q level: got %q want %q", tc.line, got.Level, tc.wantLevel)
		}
		if tc.wantSubstr != "" && !strings.Contains(got.Message, tc.wantSubstr) {
			t.Fatalf("line %q message %q should contain %q", tc.line, got.Message, tc.wantSubstr)
		}
	}
}

func TestAutoParser_malformedJSONFallsBackToText(t *testing.T) {
	t.Parallel()
	a := AutoParser{}
	line := `{"msg":broken no quotes}`
	got, err := a.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	// Text path: bare tokens may be parsed as levels; at minimum raw content preserved.
	if got.Message == "" {
		t.Fatal("expected non-empty message from text fallback")
	}
	if !strings.Contains(got.Message, "broken") {
		t.Fatalf("message %q should mention broken", got.Message)
	}
}

func TestParseUnixPrefix(t *testing.T) {
	t.Parallel()
	got, err := ParseUnixPrefix("1715000000 INFO ping")
	if err != nil {
		t.Fatal(err)
	}
	if got.Timestamp == nil || got.Timestamp.Unix() != 1715000000 {
		t.Fatalf("ts: %v", got.Timestamp)
	}
	if got.Level != "INFO" {
		t.Fatalf("level: %q", got.Level)
	}
	if !strings.Contains(got.Message, "ping") {
		t.Fatalf("message: %q", got.Message)
	}
}
