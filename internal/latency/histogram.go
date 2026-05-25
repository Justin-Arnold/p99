package latency

import (
	"math"
	"sort"
	"time"
)

const (
	defaultMaxExactValues     = 4096
	defaultSignificantFigures = 3
)

// Bucket uses an upper bound instead of a lower/width pair so saved run files
// can be read without knowing the histogram strategy that produced them.
type Bucket struct {
	UpperBoundNS int64 `json:"upper_bound_ns"`
	Count        int   `json:"count"`
}

type HistogramConfig struct {
	MaxExactValues     int
	SignificantFigures int
}

type Histogram struct {
	cfg       HistogramConfig
	count     int
	min       time.Duration
	max       time.Duration
	exact     []time.Duration
	buckets   map[int64]int
	compacted bool
}

func NewHistogram() *Histogram {
	return NewHistogramWithConfig(HistogramConfig{})
}

func NewHistogramWithConfig(cfg HistogramConfig) *Histogram {
	if cfg.MaxExactValues <= 0 {
		cfg.MaxExactValues = defaultMaxExactValues
	}
	if cfg.SignificantFigures <= 0 {
		cfg.SignificantFigures = defaultSignificantFigures
	}
	if cfg.SignificantFigures > 6 {
		// More precision creates many buckets without adding useful signal for
		// latency triage; exact values are retained before compaction anyway.
		cfg.SignificantFigures = 6
	}
	return &Histogram{cfg: cfg, exact: make([]time.Duration, 0, min(cfg.MaxExactValues, 64))}
}

func (h *Histogram) Record(d time.Duration) {
	if d < 0 {
		return
	}
	if h.count == 0 || d < h.min {
		h.min = d
	}
	if h.count == 0 || d > h.max {
		h.max = d
	}
	h.count++

	if !h.compacted && len(h.exact) < h.cfg.MaxExactValues {
		h.exact = append(h.exact, d)
		return
	}
	if !h.compacted {
		// Exact storage keeps short runs precise. Compaction only starts once a
		// run is large enough that bounded memory matters more than interpolation.
		h.compact()
	}
	h.recordBucket(d)
}

func (h *Histogram) Count() int {
	return h.count
}

func (h *Histogram) Values() []time.Duration {
	out := make([]time.Duration, len(h.exact))
	copy(out, h.exact)
	return out
}

func (h *Histogram) Percentile(p float64) time.Duration {
	if h.count == 0 {
		return 0
	}
	if p <= 0 {
		return h.min
	}
	if p >= 100 {
		return h.max
	}
	if !h.compacted {
		return exactPercentile(h.sorted(), p)
	}

	// After compaction, percentile answers are bucket upper bounds. Returning the
	// bound intentionally errs high, which is safer for latency budgets.
	rank := int(math.Ceil((p / 100) * float64(h.count)))
	if rank < 1 {
		rank = 1
	}
	var cumulative int
	for _, bucket := range h.Buckets() {
		cumulative += bucket.Count
		if cumulative >= rank {
			if bucket.UpperBoundNS < 0 {
				return h.max
			}
			return time.Duration(bucket.UpperBoundNS)
		}
	}
	return h.max
}

func (h *Histogram) Buckets() []Bucket {
	if h.count == 0 {
		return nil
	}
	if !h.compacted {
		counts := map[int64]int{}
		for _, value := range h.exact {
			// Run files use the same bucket representation for exact and compacted
			// runs so report/export code does not need two distribution formats.
			counts[roundUpSignificant(value.Nanoseconds(), h.cfg.SignificantFigures)]++
		}
		return sortedBuckets(counts)
	}
	return sortedBuckets(h.buckets)
}

func (h *Histogram) Min() time.Duration {
	return h.min
}

func (h *Histogram) Max() time.Duration {
	return h.max
}

func (h *Histogram) IsCompacted() bool {
	return h.compacted
}

func (h *Histogram) ExactValueCount() int {
	return len(h.exact)
}

func (h *Histogram) BucketCount() int {
	if !h.compacted {
		return len(h.Buckets())
	}
	return len(h.buckets)
}

func (h *Histogram) compact() {
	h.buckets = map[int64]int{}
	for _, value := range h.exact {
		h.recordBucket(value)
	}
	h.exact = nil
	h.compacted = true
}

func (h *Histogram) recordBucket(d time.Duration) {
	if h.buckets == nil {
		h.buckets = map[int64]int{}
	}
	h.buckets[roundUpSignificant(d.Nanoseconds(), h.cfg.SignificantFigures)]++
}

func (h *Histogram) sorted() []time.Duration {
	values := h.Values()
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func exactPercentile(sorted []time.Duration, p float64) time.Duration {
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
	if frac == 0 {
		return sorted[i]
	}
	lower := float64(sorted[i])
	upper := float64(sorted[i+1])
	return time.Duration(lower + (upper-lower)*frac)
}

func sortedBuckets(counts map[int64]int) []Bucket {
	keys := make([]int64, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]Bucket, 0, len(keys))
	for _, key := range keys {
		out = append(out, Bucket{UpperBoundNS: key, Count: counts[key]})
	}
	return out
}

func roundUpSignificant(value int64, significantFigures int) int64 {
	if value <= 0 {
		return 0
	}
	if significantFigures <= 0 {
		significantFigures = defaultSignificantFigures
	}
	digits := int(math.Floor(math.Log10(float64(value)))) + 1
	scalePower := digits - significantFigures
	if scalePower <= 0 {
		return value
	}
	scale := int64(math.Pow10(scalePower))
	// Rounding up keeps bucket labels as "at or below" bounds instead of
	// nearest representatives, which is easier to reason about in reports.
	return ((value + scale - 1) / scale) * scale
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
