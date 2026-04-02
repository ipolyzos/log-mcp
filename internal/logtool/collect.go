package logtool

import (
	"context"
	"time"

	"github.com/ipolyzos/log-mcp/internal/normalize"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// Line is one redacted log row suitable for JSON tool output.
type Line struct {
	LineNumber int64
	Timestamp  *time.Time
	Level      string
	Message    string
	Text       string // full redacted line
}

// Collect runs the streaming pipeline and returns up to maxOut lines that
// match filter. When the limit is hit, the context is cancelled so the source
// stops reading; remaining channel values are drained.
func Collect(ctx context.Context, src source.LogSource, q source.Query, f Filter, maxOut int64) ([]Line, bool, error) {
	if maxOut <= 0 {
		maxOut = defaultMaxOut
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pl := &pipeline.Pipeline{
		Source: src,
		Query:  q,
		Config: normalize.DefaultConfig(),
	}
	out, err := pl.Run(ctx)
	if err != nil {
		return nil, false, err
	}

	var lines []Line
	limitHit := false
	for r := range out {
		if !Match(f, r) {
			continue
		}
		if int64(len(lines)) < maxOut {
			lines = append(lines, toLine(r))
			continue
		}
		if !limitHit {
			limitHit = true
			cancel()
		}
	}
	return lines, limitHit, nil
}

func toLine(r pipeline.Result) Line {
	return Line{
		LineNumber: r.LineNumber,
		Timestamp:  r.Parsed.Timestamp,
		Level:      r.Parsed.Level,
		Message:    r.Parsed.Message,
		Text:       r.Redacted,
	}
}

const defaultMaxOut = 500
