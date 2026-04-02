package normalize

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	rfc3339LikeRE = regexp.MustCompile(
		`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})?`,
	)

	unixSecondsRE = regexp.MustCompile(`\b[0-9]{10}\b`)
)

var tsLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.000",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.000",
}

// canonicalTime formats UTC times in a single fixed layout so the same instant
// always produces the same string (deterministic grouping / tests).
func canonicalTime(t time.Time) string {
	u := t.UTC()
	ns := u.Nanosecond()
	if ns == 0 {
		return u.Format("2006-01-02T15:04:05Z")
	}
	// Fixed-width fractional seconds (nanoseconds).
	return u.Format("2006-01-02T15:04:05.000000000Z")
}

func normalizeTimestamps(s string) string {
	s = rfc3339LikeRE.ReplaceAllStringFunc(s, func(m string) string {
		trim := strings.TrimSpace(m)
		for _, layout := range tsLayouts {
			if parsed, err := time.Parse(layout, trim); err == nil {
				return canonicalTime(parsed)
			}
		}
		if strings.Contains(trim, " ") && !strings.Contains(trim, "T") {
			t2 := strings.Replace(trim, " ", "T", 1)
			for _, layout := range tsLayouts {
				if parsed, err := time.Parse(layout, t2); err == nil {
					return canonicalTime(parsed)
				}
			}
		}
		return m
	})

	s = unixSecondsRE.ReplaceAllStringFunc(s, func(m string) string {
		sec, err := strconv.ParseInt(m, 10, 64)
		if err != nil {
			return m
		}
		t := time.Unix(sec, 0).UTC()
		if t.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) ||
			t.After(time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)) {
			return m
		}
		return canonicalTime(t)
	})

	return s
}
