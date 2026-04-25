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
	if buckets[0].Count != 1 {
		t.Fatalf("first bucket count got %d, want 1", buckets[0].Count)
	}
	var total int
	for _, b := range buckets {
		total += b.Count
	}
	if total != 2 {
		t.Fatalf("bucket total got %d, want 2", total)
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
