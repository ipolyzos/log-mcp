package parse

import "testing"

func BenchmarkJSONParser_Parse(b *testing.B) {
	line := `{"@timestamp":"2024-06-01T12:00:00Z","level":"info","msg":"hello","trace_id":"abc","n":42}`
	var p JSONParser
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.Parse(line)
	}
}

func BenchmarkTextParser_Parse(b *testing.B) {
	line := `2024-06-01T12:00:00Z INFO hello world`
	p := TextParser{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = p.Parse(line)
	}
}
