package parse

import "time"

// ParsedLine is a structured view of one log line after parsing.
type ParsedLine struct {
	Timestamp *time.Time
	Level     string
	Message   string
	Fields    map[string]string
}
