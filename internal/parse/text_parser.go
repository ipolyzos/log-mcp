package parse

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TextParser extracts structure from common plain-text log lines using
// lightweight heuristics (timestamp prefix, bracketed or bare level, key=value pairs).
type TextParser struct{}

var (
	textRFC3339Prefix = regexp.MustCompile(
		`^\s*(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)\s+`,
	)
	// Kubernetes CRI log line: stream (stdout|stderr), tag (F=full, P=partial), then message.
	// Ref: https://kubernetes.io/docs/concepts/cluster-administration/logging/
	textCRIPrefix    = regexp.MustCompile(`(?i)^(stdout|stderr)\s+([FP])\s+(.*)$`)
	textBracketLevel = regexp.MustCompile(`\[(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\]`)
	textBareLevel    = regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\b`)
	textKV           = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)=("([^"]*)"|'([^']*)'|(\S+))`)
	squashSpaceRE    = regexp.MustCompile(`\s+`)
)

// stderrErrorNeedles are matched case-insensitively on the post-CRI body when the stream
// is stderr and no explicit log level was found, to infer ERROR for operator-relevant lines.
var stderrErrorNeedles = []string{
	"connection refused",
	"connection reset",
	"no route to host",
	"network is unreachable",
	"broken pipe",
	"timed out",
	"timeout",
	"i/o timeout",
	"deadline exceeded",
	"context deadline",
	"fatal",
	"panic:",
	"panic ",
	"exception",
	"error:",
	"error=",
	"failed",
	"cannot ",
	"can't ",
	"unable to",
	"permission denied",
	"access denied",
	"unauthorized",
	"forbidden",
	"denied",
	"out of memory",
	"segfault",
	"stack trace",
	"internal server error",
}

// Parse always succeeds: at minimum the trimmed line becomes the message when
// no structure is detected.
func (TextParser) Parse(line string) (ParsedLine, error) {
	s := strings.TrimSpace(line)
	if s == "" {
		return ParsedLine{}, nil
	}
	work := s
	var out ParsedLine

	if m := textRFC3339Prefix.FindStringSubmatch(work); m != nil {
		if ts, ok := parseTimeString(m[1]); ok {
			out.Timestamp = &ts
			work = strings.TrimSpace(work[len(m[0]):])
		}
	}

	var criStream string
	var criStderr bool
	if m := textCRIPrefix.FindStringSubmatch(strings.TrimSpace(work)); m != nil {
		criStream = strings.ToLower(m[1])
		work = strings.TrimSpace(m[3])
		criStderr = criStream == "stderr"
	}

	if loc := textBracketLevel.FindStringIndex(work); loc != nil {
		sub := textBracketLevel.FindStringSubmatch(work)
		if len(sub) > 1 {
			out.Level = normalizeLevel(sub[1])
			work = strings.TrimSpace(work[:loc[0]] + work[loc[1]:])
		}
	} else if loc := textBareLevel.FindStringSubmatchIndex(work); loc != nil {
		out.Level = normalizeLevel(work[loc[2]:loc[3]])
		work = strings.TrimSpace(work[:loc[0]] + work[loc[1]:])
	}

	if criStderr && out.Level == "" && stderrImpliesError(work) {
		out.Level = "ERROR"
	}

	fields := collectKV(work)
	for k, v := range fields {
		work = removeKVSegment(work, k, v)
	}
	work = squashSpaces(work)
	if len(fields) > 0 {
		out.Fields = fields
	}

	if work != "" {
		out.Message = work
	} else {
		out.Message = s
	}
	if criStream != "" {
		if out.Fields == nil {
			out.Fields = make(map[string]string)
		}
		if _, dup := out.Fields["cri_stream"]; !dup {
			out.Fields["cri_stream"] = criStream
		}
	}
	return out, nil
}

func stderrImpliesError(msg string) bool {
	low := strings.ToLower(msg)
	for _, needle := range stderrErrorNeedles {
		if strings.Contains(low, needle) {
			return true
		}
	}
	return false
}

func normalizeLevel(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	if u == "WARNING" {
		return "WARN"
	}
	return u
}

func collectKV(s string) map[string]string {
	matches := textKV.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, m := range matches {
		key := m[1]
		var val string
		switch {
		case m[3] != "":
			val = m[3]
		case m[4] != "":
			val = m[4]
		case m[5] != "":
			val = m[5]
		default:
			continue
		}
		if _, dup := out[key]; !dup {
			out[key] = val
		}
	}
	return out
}

func removeKVSegment(s, key, val string) string {
	candidates := []string{
		key + `="` + val + `"`,
		key + `='` + val + `'`,
		key + "=" + val,
	}
	for _, p := range candidates {
		if i := strings.Index(s, p); i >= 0 {
			s = s[:i] + s[i+len(p):]
		}
	}
	return s
}

func squashSpaces(s string) string {
	return strings.TrimSpace(squashSpaceRE.ReplaceAllString(s, " "))
}

// AutoParser tries [JSONParser] when the trimmed line starts with '{';
// malformed JSON falls back to [TextParser].
type AutoParser struct {
	JSON JSONParser
	Text TextParser
}

// Parse handles mixed JSON / text log files.
func (a AutoParser) Parse(line string) (ParsedLine, error) {
	s := strings.TrimSpace(line)
	if len(s) > 0 && s[0] == '{' {
		if p, err := a.JSON.Parse(line); err == nil {
			return p, nil
		}
	}
	return a.Text.Parse(line)
}

// ParseUnixPrefix parses a leading decimal unix-seconds field, then applies
// [TextParser] to the remainder (level, key=value, etc.).
func ParseUnixPrefix(line string) (ParsedLine, error) {
	s := strings.TrimSpace(line)
	i := strings.IndexByte(s, ' ')
	if i <= 0 {
		return TextParser{}.Parse(line)
	}
	sec, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil || sec <= 0 {
		return TextParser{}.Parse(line)
	}
	ts := time.Unix(sec, 0)
	rest := strings.TrimSpace(s[i+1:])
	inner, _ := TextParser{}.Parse(rest)
	p := ParsedLine{Timestamp: &ts, Message: inner.Message, Level: inner.Level}
	if len(inner.Fields) > 0 {
		p.Fields = inner.Fields
	}
	if p.Message == "" {
		p.Message = rest
	}
	return p, nil
}
