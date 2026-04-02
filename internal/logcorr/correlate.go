package logcorr

import (
	"context"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/source"
)

// CorrelateResult is the timeline of events sharing one field value.
type CorrelateResult struct {
	StreamID     string
	Field        string
	Value        string
	ScannedLines int64
	Truncated    bool
	Events       []LineEvent
}

const (
	defaultMaxScanCorr = int64(20000)
	defaultMaxEvents   = int64(500)
	maxEventsCap       = int64(2000)
)

// CorrelateEvents scans up to maxScan lines and returns all lines whose logical
// field equals wantValue (exact match after TrimSpace), ordered by time then line number.
func CorrelateEvents(ctx context.Context, src source.LogSource, streamID, field, wantValue string, maxScan, maxEvents int64) (*CorrelateResult, error) {
	streamID = strings.TrimSpace(streamID)
	field = strings.TrimSpace(field)
	wantValue = strings.TrimSpace(wantValue)
	if wantValue == "" {
		return nil, ErrEmptyValue
	}
	if maxScan <= 0 {
		maxScan = defaultMaxScanCorr
	}
	if maxScan > maxScanCap {
		maxScan = maxScanCap
	}
	if maxEvents <= 0 {
		maxEvents = defaultMaxEvents
	}
	if maxEvents > maxEventsCap {
		maxEvents = maxEventsCap
	}

	q := source.Query{StreamID: streamID, MaxLines: maxScan}
	out, err := streamResults(ctx, src, q)
	if err != nil {
		return nil, err
	}

	var scanned int64
	var events []LineEvent
	for r := range out {
		scanned++
		v, ok := GetField(r.Parsed, field)
		if !ok || v != wantValue {
			continue
		}
		events = append(events, LineEvent{Result: r})
	}

	SortEvents(events)
	truncatedScan := maxScan > 0 && scanned >= maxScan
	truncatedEvents := int64(len(events)) > maxEvents
	if truncatedEvents {
		events = events[:maxEvents]
	}

	return &CorrelateResult{
		StreamID:     streamID,
		Field:        field,
		Value:        wantValue,
		ScannedLines: scanned,
		Truncated:    truncatedScan || truncatedEvents,
		Events:       events,
	}, nil
}
