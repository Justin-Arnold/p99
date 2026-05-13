package latency

import "time"

type Summary struct {
	Count     int           `json:"count"`
	Success   int           `json:"success_count"`
	Errors    int           `json:"error_count"`
	ErrorRate float64       `json:"error_rate"`
	Min       time.Duration `json:"min_ns"`
	P50       time.Duration `json:"p50_ns"`
	P90       time.Duration `json:"p90_ns"`
	P95       time.Duration `json:"p95_ns"`
	P99       time.Duration `json:"p99_ns"`
	P999      time.Duration `json:"p999_ns"`
	Max       time.Duration `json:"max_ns"`
}

func Summarize(h *Histogram, success, errors int) Summary {
	count := h.Count()
	s := Summary{
		Count:   count,
		Success: success,
		Errors:  errors,
	}
	if count > 0 {
		s.Min = h.Min()
		s.P50 = h.Percentile(50)
		s.P90 = h.Percentile(90)
		s.P95 = h.Percentile(95)
		s.P99 = h.Percentile(99)
		s.P999 = h.Percentile(99.9)
		s.Max = h.Max()
	}
	if success+errors > 0 {
		s.ErrorRate = float64(errors) / float64(success+errors)
	}
	return s
}
