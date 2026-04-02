package logtool

import (
	"testing"

	"github.com/ipolyzos/log-mcp/internal/source"
)

func TestEnrichQuery_setsCapsFromEnv(t *testing.T) {
	t.Setenv("LOG_MCP_SEC_MAX_STREAM_BYTES", "999")
	t.Setenv("LOG_MCP_SEC_MAX_LINE_BYTES", "88")
	q := source.Query{StreamID: "x"}
	EnrichQuery(&q)
	if q.MaxBytes != 999 {
		t.Fatalf("MaxBytes: got %d", q.MaxBytes)
	}
	if q.MaxLineBytes != 88 {
		t.Fatalf("MaxLineBytes: got %d", q.MaxLineBytes)
	}
}

func TestEnrichQuery_preservesExplicitLimits(t *testing.T) {
	t.Setenv("LOG_MCP_SEC_MAX_STREAM_BYTES", "999")
	q := source.Query{StreamID: "x", MaxBytes: 50, MaxLineBytes: 10}
	EnrichQuery(&q)
	if q.MaxBytes != 50 || q.MaxLineBytes != 10 {
		t.Fatalf("got MaxBytes=%d MaxLineBytes=%d", q.MaxBytes, q.MaxLineBytes)
	}
}
