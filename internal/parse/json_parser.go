package parse

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// JSONParser extracts timestamp, level, message, and remaining fields from a
// JSON object on one line.
type JSONParser struct{}

var jsonTimestampKeys = []string{
	"@timestamp", "timestamp", "time", "ts", "datetime", "date", "created", "logged_at",
}
var jsonLevelKeys = []string{
	"level", "severity", "lvl", "log_level", "priority", "log.level",
}
var jsonMessageKeys = []string{
	"msg", "message", "text", "body", "log", "message_text", "log.message",
}

// Parse unmarshals a single JSON object from line. It returns an error if the
// line is not a JSON object or JSON is invalid.
func (JSONParser) Parse(line string) (ParsedLine, error) {
	s := strings.TrimSpace(line)
	if len(s) == 0 {
		return ParsedLine{}, fmt.Errorf("parse: empty line")
	}
	if s[0] != '{' {
		return ParsedLine{}, fmt.Errorf("parse: not a json object")
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return ParsedLine{}, fmt.Errorf("parse: json: %w", err)
	}
	return extractFromMap(obj), nil
}

func extractFromMap(obj map[string]any) ParsedLine {
	used := make(map[string]struct{})
	var out ParsedLine

	for _, k := range jsonTimestampKeys {
		if v, ok := obj[k]; ok {
			if ts, ok := parseTimeValue(v); ok {
				out.Timestamp = &ts
				used[k] = struct{}{}
				break
			}
		}
	}
	for _, k := range jsonLevelKeys {
		if v, ok := obj[k]; ok {
			if s, ok := stringifyScalar(v); ok {
				out.Level = normalizeLevel(strings.TrimSpace(s))
				used[k] = struct{}{}
				break
			}
		}
	}
	for _, k := range jsonMessageKeys {
		if v, ok := obj[k]; ok {
			if s, ok := stringifyScalar(v); ok {
				out.Message = strings.TrimSpace(s)
				used[k] = struct{}{}
				break
			}
		}
	}

	fields := make(map[string]string)
	for k, v := range obj {
		if _, skip := used[k]; skip {
			continue
		}
		fields[k] = stringifyAny(v)
	}
	if len(fields) > 0 {
		out.Fields = fields
	}
	return out
}

func stringifyScalar(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case json.Number:
		return x.String(), true
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return "", false
		}
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(x), true
	case nil:
		return "", false
	default:
		return "", false
	}
}

func stringifyAny(v any) string {
	if s, ok := stringifyScalar(v); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func parseTimeValue(v any) (time.Time, bool) {
	switch x := v.(type) {
	case string:
		return parseTimeString(x)
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return time.Time{}, false
		}
		return unixFloatToTime(f)
	case float64:
		return unixFloatToTime(x)
	case int64:
		return unixIntToTime(x)
	default:
		return time.Time{}, false
	}
}

func unixIntToTime(n int64) (time.Time, bool) {
	switch {
	case n > 1_000_000_000_000_000: // microseconds
		return time.UnixMicro(n), true
	case n > 999_999_999_999: // milliseconds (>= ~2001-09-09 in ms)
		return time.UnixMilli(n), true
	default:
		return time.Unix(n, 0), true
	}
}

func unixFloatToTime(f float64) (time.Time, bool) {
	switch {
	case f > 1e15: // microseconds as float
		return time.UnixMicro(int64(f)), true
	case f > 1e12: // milliseconds
		return time.UnixMilli(int64(f)), true
	case f > 1e9: // seconds with fractional part
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), true
	default:
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), true
	}
}

func parseTimeString(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000000000Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	if t, err := time.Parse(time.RFC1123Z, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}
