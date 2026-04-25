package latency

import (
	"fmt"
	"sort"
	"time"
)

type TimedLatency struct {
	At      time.Time
	Latency time.Duration
}

type Shape struct {
	Kind  string   `json:"kind"`
	Notes []string `json:"notes,omitempty"`
	Hints []string `json:"hints,omitempty"`
}

func AnalyzeShape(points []TimedLatency) Shape {
	if len(points) < 20 {
		return Shape{Kind: "insufficient_signal", Notes: []string{"too few requests to characterize the distribution"}}
	}

	values := make([]time.Duration, 0, len(points))
	for _, p := range points {
		values = append(values, p.Latency)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })

	p50 := percentile(values, 50)
	p90 := percentile(values, 90)
	p99 := percentile(values, 99)

	if degrading(points) {
		return Shape{
			Kind:  "degrading_over_time",
			Notes: []string{"later requests are materially slower than earlier requests"},
			Hints: []string{"This can happen with queue buildup, resource exhaustion, GC pressure, or a downstream service slowing under load."},
		}
	}

	if p99 > p50*8 && p99-p90 > p50*4 {
		return Shape{
			Kind:  "spiky",
			Notes: []string{fmt.Sprintf("p99 is much higher than the median while most requests remain closer to %s", p50)},
			Hints: []string{"This can happen with retries, intermittent dependency stalls, lock contention, or noisy neighbors."},
		}
	}

	if looksBimodal(values) {
		return Shape{
			Kind:  "bimodal",
			Notes: []string{"latencies appear to cluster in two separated bands"},
			Hints: []string{"This can happen with cache misses, cold paths, connection pool waits, retries, or slow dependency calls."},
		}
	}

	if p90 < p50*3 && p99 < p50*5 {
		return Shape{
			Kind:  "stable",
			Notes: []string{"tail latency stays within a moderate multiple of the median"},
		}
	}

	return Shape{
		Kind:  "insufficient_signal",
		Notes: []string{"distribution does not match a conservative built-in pattern"},
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := (p / 100) * float64(len(sorted)-1)
	i := int(rank)
	frac := rank - float64(i)
	return time.Duration(float64(sorted[i]) + (float64(sorted[i+1])-float64(sorted[i]))*frac)
}

func degrading(points []TimedLatency) bool {
	ordered := append([]TimedLatency(nil), points...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].At.Before(ordered[j].At) })
	third := len(ordered) / 3
	if third == 0 {
		return false
	}
	first := make([]time.Duration, 0, third)
	last := make([]time.Duration, 0, third)
	for _, p := range ordered[:third] {
		first = append(first, p.Latency)
	}
	for _, p := range ordered[len(ordered)-third:] {
		last = append(last, p.Latency)
	}
	sort.Slice(first, func(i, j int) bool { return first[i] < first[j] })
	sort.Slice(last, func(i, j int) bool { return last[i] < last[j] })
	return percentile(last, 50) > percentile(first, 50)*3
}

func looksBimodal(sorted []time.Duration) bool {
	if len(sorted) < 30 {
		return false
	}
	var largestGap time.Duration
	var gapIndex int
	for i := 1; i < len(sorted); i++ {
		gap := sorted[i] - sorted[i-1]
		if gap > largestGap {
			largestGap = gap
			gapIndex = i
		}
	}
	left := gapIndex
	right := len(sorted) - gapIndex
	if left < len(sorted)/5 || right < len(sorted)/5 {
		return false
	}
	return largestGap > percentile(sorted, 50)*2
}
