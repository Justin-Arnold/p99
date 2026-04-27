package probe

import (
	"time"

	"github.com/justin/p99/internal/latency"
)

const ResultVersion = 1

type HTTPConfig struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Duration        time.Duration     `json:"duration_ns"`
	RPS             float64           `json:"rps"`
	Concurrency     int               `json:"concurrency"`
	Warmup          time.Duration     `json:"warmup_ns"`
	Timeout         time.Duration     `json:"timeout_ns"`
	Headers         map[string]string `json:"headers,omitempty"`
	BodyFile        string            `json:"body_file,omitempty"`
	RequestBodySize int               `json:"request_body_size,omitempty"`
	ExpectedStatus  []int             `json:"expected_status,omitempty"`
	SlowSamples     int               `json:"slow_samples"`
}

type RunResult struct {
	Version     int                  `json:"version"`
	Config      HTTPConfig           `json:"config"`
	StartedAt   time.Time            `json:"started_at"`
	EndedAt     time.Time            `json:"ended_at"`
	Summary     latency.Summary      `json:"summary"`
	Histogram   []latency.Bucket     `json:"histogram"`
	SlowSamples []latency.SlowSample `json:"slow_samples,omitempty"`
	Errors      map[string]int       `json:"error_breakdown,omitempty"`
	Shape       latency.Shape        `json:"shape"`
}
