package reporter

import (
	"encoding/json"
	"math"
	"time"

	"github.com/miekg/dns"
)

type jsonReporter struct{}

type latencyStats struct {
	MinMs  float64 `json:"minMs"`
	MeanMs float64 `json:"meanMs"`
	StdMs  float64 `json:"stdMs"`
	MaxMs  float64 `json:"maxMs"`
	P99Ms  float64 `json:"p99Ms"`
	P95Ms  float64 `json:"p95Ms"`
	P90Ms  float64 `json:"p90Ms"`
	P75Ms  float64 `json:"p75Ms"`
	P50Ms  float64 `json:"p50Ms"`
}

type histogramPoint struct {
	LatencyMs int64 `json:"latencyMs"`
	Count     int64 `json:"count"`
}

type jsonResult struct {
	TotalRequests              int64            `json:"totalRequests"`
	TotalSuccessResponses      int64            `json:"totalSuccessResponses"`
	TotalNegativeResponses     int64            `json:"totalNegativeResponses"`
	TotalErrorResponses        int64            `json:"totalErrorResponses"`
	TotalIOErrors              int64            `json:"totalIOErrors"`
	TotalIDmismatch            int64            `json:"totalIDmismatch"`
	TotalTruncatedResponses    int64            `json:"totalTruncatedResponses"`
	ResponseRcodes             map[string]int64 `json:"responseRcodes,omitempty"`
	QuestionTypes              map[string]int64 `json:"questionTypes"`
	QueriesPerSecond           float64          `json:"queriesPerSecond"`
	BenchmarkDurationSeconds   float64          `json:"benchmarkDurationSeconds"`
	LatencyStats               latencyStats     `json:"latencyStats"`
	LatencyDistribution        []histogramPoint `json:"latencyDistribution,omitempty"`
	TotalDNSSECSecuredDomains  *int             `json:"totalDNSSECSecuredDomains,omitempty"`
	DohHTTPResponseStatusCodes map[int]int64    `json:"dohHTTPResponseStatusCodes,omitempty"`
	ExtendedDNSErrors          map[uint16]int64 `json:"extendedDNSErrors,omitempty"`
}

func (s *jsonReporter) print(params reportParameters) error {
	codeTotalsMapped := make(map[string]int64)
	if params.benchmark.Rcodes {
		for k, v := range params.codeTotals {
			codeTotalsMapped[dns.RcodeToString[k]] = v
		}
	}

	var res []histogramPoint

	if params.benchmark.HistDisplay {
		dist := params.hist.Distribution()
		for _, d := range dist {
			res = append(res, histogramPoint{
				LatencyMs: roundDuration(time.Duration(d.To/2 + d.From/2)).Milliseconds(),
				Count:     d.Count,
			})
		}

		var dedupRes []histogramPoint
		i := -1
		for _, r := range res {
			if i >= 0 {
				if dedupRes[i].LatencyMs == r.LatencyMs {
					dedupRes[i].Count += r.Count
				} else {
					dedupRes = append(dedupRes, r)
					i++
				}
			} else {
				dedupRes = append(dedupRes, r)
				i++
			}
		}
		res = dedupRes
	}

	result := jsonResult{
		TotalRequests:            params.totalCounters.Total,
		TotalSuccessResponses:    params.totalCounters.Success,
		TotalNegativeResponses:   params.totalCounters.Negative,
		TotalErrorResponses:      params.totalCounters.Error,
		TotalIOErrors:            params.totalCounters.IOError,
		TotalIDmismatch:          params.totalCounters.IDmismatch,
		TotalTruncatedResponses:  params.totalCounters.Truncated,
		QueriesPerSecond:         math.Round(float64(params.totalCounters.Total)/params.benchmarkDuration.Seconds()*100) / 100,
		BenchmarkDurationSeconds: roundDuration(params.benchmarkDuration).Seconds(),
		ResponseRcodes:           codeTotalsMapped,
		QuestionTypes:            params.qtypeTotals,
		LatencyStats: latencyStats{
			MinMs:  float64(time.Duration(params.hist.Min()).Microseconds()) / 1000,
			MeanMs: float64(time.Duration(params.hist.Mean()).Microseconds()) / 1000,
			StdMs:  float64(time.Duration(params.hist.StdDev()).Microseconds()) / 1000,
			MaxMs:  float64(time.Duration(params.hist.Max()).Microseconds()) / 1000,
			P99Ms:  float64(time.Duration(params.hist.ValueAtQuantile(99)).Microseconds()) / 1000,
			P95Ms:  float64(time.Duration(params.hist.ValueAtQuantile(95)).Microseconds()) / 1000,
			P90Ms:  float64(time.Duration(params.hist.ValueAtQuantile(90)).Microseconds()) / 1000,
			P75Ms:  float64(time.Duration(params.hist.ValueAtQuantile(75)).Microseconds()) / 1000,
			P50Ms:  float64(time.Duration(params.hist.ValueAtQuantile(50)).Microseconds()) / 1000,
		},
		LatencyDistribution:        res,
		DohHTTPResponseStatusCodes: params.dohResponseStatusesTotals,
		ExtendedDNSErrors:          params.edeCodes,
	}
	if params.benchmark.DNSSEC {
		totalDNSSECSecuredDomains := len(params.authenticatedDomains)
		result.TotalDNSSECSecuredDomains = &totalDNSSECSecuredDomains
	}

	return json.NewEncoder(params.outputWriter).Encode(result)
}
