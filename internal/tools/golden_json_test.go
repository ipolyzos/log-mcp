package tools

import (
	"bytes"
	"encoding/json"
	"github.com/ipolyzos/log-mcp/internal/loganalysis"
	"os"
	"path/filepath"
	"testing"
)

// Golden files document stable JSON shapes for MCP structured outputs (indent
// two spaces, no trailing newline in file — we trim before compare).
func TestGoldenJSON_logsToolOutput(t *testing.T) {
	t.Parallel()
	want := readGolden(t, "logs_tool_output.json")
	v := LogsToolOutput{
		StreamID: "/tmp/x",
		Lines: []LogLineJSON{
			{LineNumber: 1, Text: "hello"},
		},
		Truncated: false,
	}
	got := mustMarshalIndent(t, v)
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGoldenJSON_detectAnomaliesOutput(t *testing.T) {
	t.Parallel()
	want := readGolden(t, "detect_anomalies_output.json")
	v := DetectAnomaliesOutput{
		StreamID:           "k8s://ns/pod",
		ScannedLines:       0,
		Truncated:          false,
		LowSample:          true,
		SegmentCount:       15,
		ActiveSegmentCount: 0,
		LevelSpikeSegments: []loganalysis.LevelSpikeSegment{},
		RepeatedErrors:     []loganalysis.RepeatedErrorBurst{},
	}
	got := mustMarshalIndent(t, v)
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("testdata", "golden", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustMarshalIndent(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	// Encoder.Encode adds a trailing newline; golden files omit it.
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
