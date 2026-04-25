package latency

import (
	"sort"
	"time"
)

type SlowSample struct {
	Latency   time.Duration `json:"latency_ns"`
	Timestamp time.Time     `json:"timestamp"`
	Status    int           `json:"status_code,omitempty"`
	Error     string        `json:"error_class,omitempty"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
}

type SlowSampler struct {
	limit   int
	samples []SlowSample
}

func NewSlowSampler(limit int) *SlowSampler {
	return &SlowSampler{limit: limit}
}

func (s *SlowSampler) Add(sample SlowSample) {
	if s.limit <= 0 {
		return
	}
	s.samples = append(s.samples, sample)
	sort.Slice(s.samples, func(i, j int) bool {
		return s.samples[i].Latency > s.samples[j].Latency
	})
	if len(s.samples) > s.limit {
		s.samples = s.samples[:s.limit]
	}
}

func (s *SlowSampler) Samples() []SlowSample {
	out := make([]SlowSample, len(s.samples))
	copy(out, s.samples)
	return out
}
