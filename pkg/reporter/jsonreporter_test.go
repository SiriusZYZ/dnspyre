package reporter

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

func Test_histogramLatencyMs(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want float64
	}{
		{name: "zero", in: 0, want: 0},
		{name: "under 100us", in: 50 * time.Microsecond, want: 0},
		{name: "100us", in: 100 * time.Microsecond, want: 0.1},
		{name: "250us", in: 250 * time.Microsecond, want: 0.2},
		{name: "999us", in: 999 * time.Microsecond, want: 0.9},
		{name: "1ms", in: time.Millisecond, want: 1},
		{name: "1.5ms", in: 1500 * time.Microsecond, want: 1},
		{name: "8ms", in: 8 * time.Millisecond, want: 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, histogramLatencyMs(tt.in))
		})
	}
}

func Test_jsonDistributionBuckets(t *testing.T) {
	h := hdrhistogram.New(0, int64(5*time.Second), 1)
	require.NoError(t, h.RecordValue(int64(50*time.Microsecond)))
	require.NoError(t, h.RecordValue(int64(250*time.Microsecond)))
	require.NoError(t, h.RecordValue(int64(250*time.Microsecond)))
	require.NoError(t, h.RecordValue(int64(2*time.Millisecond)))

	var buf bytes.Buffer
	err := (&jsonReporter{}).print(reportParameters{
		benchmark:         &dnsbench.Benchmark{HistDisplay: true},
		outputWriter:      &buf,
		hist:              h,
		totalCounters:     dnsbench.Counters{Total: 4},
		qtypeTotals:       map[string]int64{"A": 4},
		benchmarkDuration: time.Second,
	})
	require.NoError(t, err)

	var result jsonResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	for _, p := range result.LatencyDistribution {
		if p.LatencyMs < 1 {
			assert.Equal(t, math.Round(p.LatencyMs*10)/10, p.LatencyMs)
		} else {
			assert.Equal(t, float64(int64(p.LatencyMs)), p.LatencyMs)
		}
	}
	assert.Equal(t, int64(1), countAt(result.LatencyDistribution, 0))
	assert.Equal(t, int64(2), countAt(result.LatencyDistribution, 0.2))
	assert.Equal(t, int64(1), countAt(result.LatencyDistribution, 2))
}

func countAt(points []histogramPoint, ms float64) int64 {
	var n int64
	for _, p := range points {
		if math.Abs(p.LatencyMs-ms) < 1e-9 {
			n += p.Count
		}
	}
	return n
}
