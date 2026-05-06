package spans

import (
	"time"

	"github.com/justin/p99/internal/latency"
)

const ReportVersion = 1

type Span struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind,omitempty"`
	Service      string            `json:"service,omitempty"`
	Start        time.Time         `json:"start"`
	End          time.Time         `json:"end"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	StatusCode   string            `json:"status_code,omitempty"`
	StatusText   string            `json:"status_text,omitempty"`
}

func (s Span) Duration() time.Duration {
	if s.End.Before(s.Start) {
		return 0
	}
	return s.End.Sub(s.Start)
}

type Report struct {
	Version      int                 `json:"version"`
	Source       string              `json:"source,omitempty"`
	TraceCount   int                 `json:"trace_count"`
	SpanCount    int                 `json:"span_count"`
	RequestCount int                 `json:"request_count"`
	Summary      latency.Summary     `json:"summary"`
	Routes       []GroupSummary      `json:"routes,omitempty"`
	Services     []GroupSummary      `json:"services,omitempty"`
	Dependencies []DependencySummary `json:"dependencies,omitempty"`
	SlowTraces   []SlowTrace         `json:"slow_traces,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
	Hints        []string            `json:"hints,omitempty"`
}

type GroupSummary struct {
	Name    string          `json:"name"`
	Count   int             `json:"count"`
	Summary latency.Summary `json:"summary"`
}

type DependencySummary struct {
	Name            string          `json:"name"`
	Type            string          `json:"type,omitempty"`
	Service         string          `json:"service,omitempty"`
	Count           int             `json:"count"`
	Total           time.Duration   `json:"total_ns"`
	Summary         latency.Summary `json:"summary"`
	ShareOfRequests float64         `json:"share_of_request_time"`
}

type SlowTrace struct {
	TraceID         string             `json:"trace_id"`
	Route           string             `json:"route,omitempty"`
	Service         string             `json:"service,omitempty"`
	RootSpan        string             `json:"root_span"`
	Latency         time.Duration      `json:"latency_ns"`
	Start           time.Time          `json:"start"`
	DependencyTime  time.Duration      `json:"dependency_time_ns"`
	TopDependencies []DependencySample `json:"top_dependencies,omitempty"`
	Error           bool               `json:"error"`
}

type DependencySample struct {
	Name     string        `json:"name"`
	Type     string        `json:"type,omitempty"`
	Duration time.Duration `json:"duration_ns"`
}
