package latency

import (
	"testing"
	"time"
)

func TestPercentileInterpolates(t *testing.T) {
	h := NewHistogram()
	for _, v := range []time.Duration{10, 20, 30, 40} {
		h.Record(v * time.Millisecond)
	}
	if got := h.Percentile(50); got != 25*time.Millisecond {
		t.Fatalf("p50 got %s, want 25ms", got)
	}
	if got := h.Percentile(100); got != 40*time.Millisecond {
		t.Fatalf("max percentile got %s, want 40ms", got)
	}
}

func TestHistogramBuckets(t *testing.T) {
	h := NewHistogram()
	h.Record(500 * time.Microsecond)
	h.Record(1500 * time.Millisecond)
	buckets := h.Buckets()
	if buckets[0].UpperBoundNS != int64(500*time.Microsecond) || buckets[0].Count != 1 {
		t.Fatalf("first bucket got %#v, want 500us count 1", buckets[0])
	}
	var total int
	for _, b := range buckets {
		total += b.Count
	}
	if total != 2 {
		t.Fatalf("bucket total got %d, want 2", total)
	}
}

func TestHistogramCompactsAfterExactLimit(t *testing.T) {
	h := NewHistogramWithConfig(HistogramConfig{MaxExactValues: 4, SignificantFigures: 3})
	for i := 1; i <= 10; i++ {
		h.Record(time.Duration(i) * time.Millisecond)
	}
	if !h.IsCompacted() {
		t.Fatal("expected histogram to compact")
	}
	if h.ExactValueCount() != 0 {
		t.Fatalf("exact value count got %d, want 0", h.ExactValueCount())
	}
	if h.Count() != 10 {
		t.Fatalf("count got %d, want 10", h.Count())
	}
	if h.Min() != time.Millisecond || h.Max() != 10*time.Millisecond {
		t.Fatalf("min/max got %s/%s", h.Min(), h.Max())
	}
}

func TestCompactedHistogramPercentileApproximation(t *testing.T) {
	h := NewHistogramWithConfig(HistogramConfig{MaxExactValues: 10, SignificantFigures: 3})
	for i := 1; i <= 1000; i++ {
		h.Record(time.Duration(i) * time.Millisecond)
	}
	got := h.Percentile(99)
	if got < 990*time.Millisecond || got > time.Second {
		t.Fatalf("p99 got %s, want within 990ms..1s", got)
	}
}

func TestSlowSamplerKeepsSlowest(t *testing.T) {
	s := NewSlowSampler(2)
	s.Add(SlowSample{Latency: 10 * time.Millisecond})
	s.Add(SlowSample{Latency: 50 * time.Millisecond})
	s.Add(SlowSample{Latency: 20 * time.Millisecond})
	got := s.Samples()
	if len(got) != 2 {
		t.Fatalf("len got %d, want 2", len(got))
	}
	if got[0].Latency != 50*time.Millisecond || got[1].Latency != 20*time.Millisecond {
		t.Fatalf("unexpected samples: %#v", got)
	}
}
