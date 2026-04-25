package latency

import (
	"sort"
	"time"
)

type Bucket struct {
	UpperBoundNS int64 `json:"upper_bound_ns"`
	Count        int   `json:"count"`
}

type Histogram struct {
	values []time.Duration
}

func NewHistogram() *Histogram {
	return &Histogram{}
}

func (h *Histogram) Record(d time.Duration) {
	if d < 0 {
		return
	}
	h.values = append(h.values, d)
}

func (h *Histogram) Count() int {
	return len(h.values)
}

func (h *Histogram) Values() []time.Duration {
	out := make([]time.Duration, len(h.values))
	copy(out, h.values)
	return out
}

func (h *Histogram) Percentile(p float64) time.Duration {
	if len(h.values) == 0 {
		return 0
	}
	values := h.sorted()
	if p <= 0 {
		return values[0]
	}
	if p >= 100 {
		return values[len(values)-1]
	}
	rank := (p / 100) * float64(len(values)-1)
	i := int(rank)
	frac := rank - float64(i)
	if frac == 0 {
		return values[i]
	}
	lower := float64(values[i])
	upper := float64(values[i+1])
	return time.Duration(lower + (upper-lower)*frac)
}

func (h *Histogram) Buckets() []Bucket {
	if len(h.values) == 0 {
		return nil
	}
	bounds := []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
	}
	buckets := make([]Bucket, len(bounds)+1)
	for i, bound := range bounds {
		buckets[i].UpperBoundNS = bound.Nanoseconds()
	}
	buckets[len(buckets)-1].UpperBoundNS = -1

	for _, v := range h.values {
		placed := false
		for i, bound := range bounds {
			if v <= bound {
				buckets[i].Count++
				placed = true
				break
			}
		}
		if !placed {
			buckets[len(buckets)-1].Count++
		}
	}
	return buckets
}

func (h *Histogram) sorted() []time.Duration {
	values := h.Values()
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}
