package security

import (
	"os"
	"strconv"
	"strings"
)

// Limits are hard caps for tool output and per-read I/O from log sources.
// Values come from the environment with safe defaults.
type Limits struct {
	// MaxToolOutputLines caps lines returned by read/search/filter and similar tools.
	MaxToolOutputLines int64
	// MaxStreamBytes caps total raw payload bytes per source.Read (excluding newlines framing overhead in counting).
	MaxStreamBytes int64
	// MaxLineBytes caps each raw line length in bytes before redaction.
	MaxLineBytes int64
	// MaxScanLines caps summarize_errors / correlation scan depth (upper bound).
	MaxScanLines int64
}

const (
	defaultMaxToolOutputLines = int64(10_000)
	defaultMaxStreamBytes     = int64(50 * 1024 * 1024) // 50 MiB
	defaultMaxLineBytes       = int64(1024 * 1024)      // 1 MiB per line
	defaultMaxScanLines       = int64(100_000)
)

// LimitsFromEnv reads LOG_MCP_SEC_* variables. Missing or invalid values fall back to defaults.
func LimitsFromEnv() Limits {
	return Limits{
		MaxToolOutputLines: parseEnvInt64("LOG_MCP_SEC_MAX_OUTPUT_LINES", defaultMaxToolOutputLines),
		MaxStreamBytes:     parseEnvInt64("LOG_MCP_SEC_MAX_STREAM_BYTES", defaultMaxStreamBytes),
		MaxLineBytes:       parseEnvInt64("LOG_MCP_SEC_MAX_LINE_BYTES", defaultMaxLineBytes),
		MaxScanLines:       parseEnvInt64("LOG_MCP_SEC_MAX_SCAN_LINES", defaultMaxScanLines),
	}
}

func parseEnvInt64(key string, def int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
