package normalize

import "regexp"

var (
	// Snowflake-like decimal runs (distinct from 10-digit unix normalization window).
	snowflakeRE = regexp.MustCompile(`\b\d{15,22}\b`)

	// Standalone hex tokens (commits, hashes); UUIDs are masked earlier in the pipeline.
	hexIDRE = regexp.MustCompile(`(?i)\b[a-f0-9]{16,40}\b`)
)

const (
	placeholderNumID = "<num_id>"
	placeholderHexID = "<hex_id>"
)

func normalizeDynamicIDs(s string) string {
	s = snowflakeRE.ReplaceAllString(s, placeholderNumID)
	s = hexIDRE.ReplaceAllString(s, placeholderHexID)
	return s
}
