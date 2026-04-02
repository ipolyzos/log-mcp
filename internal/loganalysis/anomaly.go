package loganalysis

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	"github.com/ipolyzos/log-mcp/internal/logcorr"
	"github.com/ipolyzos/log-mcp/internal/security"
	"github.com/ipolyzos/log-mcp/internal/source"
)

const (
	defaultAnomalyScan   = int64(20_000)
	defaultSegmentCount  = 15
	defaultZThreshold    = 2.0
	defaultMinRepeatErr  = int64(6)
	maxLevelSpikeReports = 30
	maxBurstReports      = 25
)

// AnomalyOptions configures [DetectAnomalies].
type AnomalyOptions struct {
	MaxScanLines    int64
	SegmentCount    int
	ZScoreThreshold float64
	MinRepeatErrors int64
}

// AnomalyReport summarizes statistically unusual regions and repeated errors.
type AnomalyReport struct {
	StreamID           string
	ScannedLines       int64
	Truncated          bool
	LevelSpikeSegments []LevelSpikeSegment
	RepeatedErrors     []RepeatedErrorBurst
}

// LevelSpikeSegment is a contiguous slice of the scan with unusually many error-level lines vs other slices.
type LevelSpikeSegment struct {
	SegmentIndex int     `json:"segment_index"`
	LineStart    int64   `json:"line_start"`
	LineEnd      int64   `json:"line_end"`
	ErrorCount   int64   `json:"error_count"`
	MeanErrors   float64 `json:"mean_errors"`
	StdDev       float64 `json:"std_dev"`
	ZScore       float64 `json:"z_score"`
}

// RepeatedErrorBurst is a redacted line pattern that appears many times at ERROR level.
type RepeatedErrorBurst struct {
	Fingerprint   string `json:"fingerprint"`
	SampleMessage string `json:"sample_message"`
	Count         int64  `json:"count"`
	FirstLine     int64  `json:"first_line"`
	LastLine      int64  `json:"last_line"`
}

type segmentAgg struct {
	errCount int64
	lineMin  int64
	lineMax  int64
}

type burstAgg struct {
	count   int64
	firstLn int64
	lastLn  int64
	sample  string
}

// DetectAnomalies scans redacted logs, buckets lines into equal-width segments by line number,
// flags segments whose ERROR/FATAL/CRITICAL count exceeds the global mean by ZScoreThreshold
// standard deviations, and reports highly repeated ERROR-level fingerprints.
func DetectAnomalies(ctx context.Context, src source.LogSource, streamID string, opts AnomalyOptions) (*AnomalyReport, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return nil, fmt.Errorf("loganalysis: empty stream id")
	}
	maxScan := opts.MaxScanLines
	if maxScan <= 0 {
		maxScan = defaultAnomalyScan
	}
	lim := security.LimitsFromEnv()
	if maxScan > lim.MaxScanLines {
		maxScan = lim.MaxScanLines
	}
	segN := opts.SegmentCount
	if segN < 2 {
		segN = defaultSegmentCount
	}
	zTh := opts.ZScoreThreshold
	if zTh <= 0 {
		zTh = defaultZThreshold
	}
	minRep := opts.MinRepeatErrors
	if minRep <= 0 {
		minRep = defaultMinRepeatErr
	}

	width := (maxScan + int64(segN) - 1) / int64(segN)
	if width < 1 {
		width = 1
	}
	segs := make([]segmentAgg, segN)
	for i := range segs {
		segs[i].lineMin = math.MaxInt64
	}

	q := source.Query{StreamID: streamID, MaxLines: maxScan}
	out, err := logcorr.StreamPipeline(ctx, src, q)
	if err != nil {
		return nil, err
	}

	bursts := make(map[uint64]*burstAgg)
	var scanned int64
	for r := range out {
		scanned++
		ln := r.LineNumber
		si := int((ln - 1) / width)
		if si >= segN {
			si = segN - 1
		}
		if isErrorLevel(r.Parsed.Level) {
			segs[si].errCount++
			h := lineHash(r.Redacted)
			b := bursts[h]
			if b == nil {
				b = &burstAgg{firstLn: ln, sample: sampleMessage(r.Parsed, r.Redacted)}
				bursts[h] = b
			}
			b.count++
			if b.firstLn == 0 || ln < b.firstLn {
				b.firstLn = ln
			}
			if ln > b.lastLn {
				b.lastLn = ln
			}
		}
		if ln < segs[si].lineMin {
			segs[si].lineMin = ln
		}
		if ln > segs[si].lineMax {
			segs[si].lineMax = ln
		}
	}

	counts := make([]float64, segN)
	for i := range segs {
		counts[i] = float64(segs[i].errCount)
	}
	mean, std := meanStdDev(counts)

	var spikes []LevelSpikeSegment
	for i, s := range segs {
		if s.errCount == 0 && s.lineMin == math.MaxInt64 {
			continue
		}
		var z float64
		if std > 1e-9 {
			z = (float64(s.errCount) - mean) / std
		} else if float64(s.errCount) > mean && s.errCount > 0 {
			z = 99
		}
		if z >= zTh && s.errCount > 0 {
			ls := s.lineMin
			le := s.lineMax
			if ls == math.MaxInt64 {
				ls = 0
			}
			spikes = append(spikes, LevelSpikeSegment{
				SegmentIndex: i,
				LineStart:    ls,
				LineEnd:      le,
				ErrorCount:   s.errCount,
				MeanErrors:   mean,
				StdDev:       std,
				ZScore:       z,
			})
		}
	}
	if len(spikes) > maxLevelSpikeReports {
		spikes = spikes[:maxLevelSpikeReports]
	}

	var rep []RepeatedErrorBurst
	for h, b := range bursts {
		if b.count < minRep {
			continue
		}
		rep = append(rep, RepeatedErrorBurst{
			Fingerprint:   fmt.Sprintf("%016x", h),
			SampleMessage: b.sample,
			Count:         b.count,
			FirstLine:     b.firstLn,
			LastLine:      b.lastLn,
		})
	}
	sort.Slice(rep, func(i, j int) bool {
		if rep[i].Count != rep[j].Count {
			return rep[i].Count > rep[j].Count
		}
		return rep[i].Fingerprint < rep[j].Fingerprint
	})
	if len(rep) > maxBurstReports {
		rep = rep[:maxBurstReports]
	}

	trunc := maxScan > 0 && scanned >= maxScan
	return &AnomalyReport{
		StreamID:           streamID,
		ScannedLines:       scanned,
		Truncated:          trunc,
		LevelSpikeSegments: spikes,
		RepeatedErrors:     rep,
	}, nil
}

func isErrorLevel(lvl string) bool {
	switch strings.ToUpper(strings.TrimSpace(lvl)) {
	case "ERROR", "ERR", "FATAL", "CRITICAL", "PANIC", "SEVERE":
		return true
	default:
		return false
	}
}

func lineHash(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.TrimSpace(s)))
	return h.Sum64()
}

func meanStdDev(xs []float64) (mean, std float64) {
	n := len(xs)
	if n == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean = sum / float64(n)
	if n == 1 {
		return mean, 0
	}
	var v float64
	for _, x := range xs {
		d := x - mean
		v += d * d
	}
	std = math.Sqrt(v / float64(n))
	return mean, std
}
