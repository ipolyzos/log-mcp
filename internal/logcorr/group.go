package logcorr

import (
	"context"
	"sort"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/security"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// GroupRow is one distinct field value bucket after scanning.
type GroupRow struct {
	Value       string
	Count       int64
	FirstLine   int64
	LastLine    int64
	LineNumbers []int64
}

// GroupByFieldResult is the outcome of grouping by a logical field.
type GroupByFieldResult struct {
	StreamID     string
	Field        string
	ScannedLines int64
	Truncated    bool
	Groups       []GroupRow
}

const (
	defaultMaxScanGroup = int64(20000)
	defaultMaxGroups    = 50
	maxGroupsCap        = 500
	maxLineNumsPerGroup = 12
)

// GroupByField scans up to maxScan lines, groups by the given logical field
// (trace_id, request_id, or a custom JSON key). Lines without the field are
// skipped. Groups are sorted by count descending, then value ascending.
func GroupByField(ctx context.Context, src source.LogSource, streamID, field string, maxScan, maxGroups int64) (*GroupByFieldResult, error) {
	streamID = strings.TrimSpace(streamID)
	field = strings.TrimSpace(field)
	if maxScan <= 0 {
		maxScan = defaultMaxScanGroup
	}
	lim := security.LimitsFromEnv()
	if maxScan > lim.MaxScanLines {
		maxScan = lim.MaxScanLines
	}
	if maxGroups <= 0 {
		maxGroups = defaultMaxGroups
	}
	if maxGroups > maxGroupsCap {
		maxGroups = maxGroupsCap
	}

	q := source.Query{StreamID: streamID, MaxLines: maxScan}
	out, err := streamResults(ctx, src, q)
	if err != nil {
		return nil, err
	}

	type acc struct {
		count    int64
		first    int64
		last     int64
		lineNums []int64
	}
	m := make(map[string]*acc)

	var scanned int64
	for r := range out {
		scanned++
		v, ok := GetField(r.Parsed, field)
		if !ok {
			continue
		}
		a, ok := m[v]
		if !ok {
			a = &acc{first: r.LineNumber, last: r.LineNumber}
			m[v] = a
		}
		a.count++
		if a.first == 0 || r.LineNumber < a.first {
			a.first = r.LineNumber
		}
		if r.LineNumber > a.last {
			a.last = r.LineNumber
		}
		if len(a.lineNums) < maxLineNumsPerGroup {
			a.lineNums = append(a.lineNums, r.LineNumber)
		}
	}

	type pair struct {
		val string
		a   *acc
	}
	list := make([]pair, 0, len(m))
	for val, a := range m {
		list = append(list, pair{val: val, a: a})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].a.count != list[j].a.count {
			return list[i].a.count > list[j].a.count
		}
		return list[i].val < list[j].val
	})
	if int64(len(list)) > maxGroups {
		list = list[:maxGroups]
	}

	rows := make([]GroupRow, 0, len(list))
	for _, p := range list {
		ln := append([]int64(nil), p.a.lineNums...)
		sort.Slice(ln, func(i, j int) bool { return ln[i] < ln[j] })
		rows = append(rows, GroupRow{
			Value:       p.val,
			Count:       p.a.count,
			FirstLine:   p.a.first,
			LastLine:    p.a.last,
			LineNumbers: ln,
		})
	}

	truncated := maxScan > 0 && scanned >= maxScan
	return &GroupByFieldResult{
		StreamID:     streamID,
		Field:        field,
		ScannedLines: scanned,
		Truncated:    truncated,
		Groups:       rows,
	}, nil
}

// LineEvent is one correlated line for sorting and export.
type LineEvent struct {
	Result pipeline.Result
}

// SortEvents orders by timestamp when both present, else by line number.
func SortEvents(ev []LineEvent) {
	sort.Slice(ev, func(i, j int) bool {
		ti, tj := ev[i].Result.Parsed.Timestamp, ev[j].Result.Parsed.Timestamp
		if ti != nil && tj != nil {
			if !ti.Equal(*tj) {
				return ti.Before(*tj)
			}
		}
		return ev[i].Result.LineNumber < ev[j].Result.LineNumber
	})
}
