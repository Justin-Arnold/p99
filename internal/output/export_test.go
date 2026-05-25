package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Justin-Arnold/p99/internal/latency"
	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/spans"
)

func TestRunPrometheusExport(t *testing.T) {
	var buf bytes.Buffer
	WritePrometheus(&buf, probe.RunResult{
		Summary: latency.Summary{Count: 3, Errors: 1, ErrorRate: 1.0 / 3.0, P99: 120 * time.Millisecond},
		Errors:  map[string]int{"HTTP_5xx": 1},
		Histogram: []latency.Bucket{
			{UpperBoundNS: int64(time.Millisecond), Count: 1},
			{UpperBoundNS: -1, Count: 2},
		},
	})
	got := buf.String()
	if !strings.Contains(got, "p99_http_requests_total 3") {
		t.Fatalf("missing request metric: %s", got)
	}
	if !strings.Contains(got, `p99_http_errors_by_class_total{class="HTTP_5xx"} 1`) {
		t.Fatalf("missing error class metric: %s", got)
	}
	if strings.Contains(got, "# TYPE p99_http_latency_seconds_bucket histogram") {
		t.Fatalf("export should not claim a Prometheus histogram without sum/count: %s", got)
	}
	if !strings.Contains(got, `p99_http_latency_quantile_seconds{quantile="0.99"} 0.12`) {
		t.Fatalf("missing quantile metric: %s", got)
	}
	if !strings.Contains(got, `p99_http_latency_bucket_requests{le="+Inf"} 3`) {
		t.Fatalf("missing cumulative bucket: %s", got)
	}
}

func TestRunOTelExport(t *testing.T) {
	var buf bytes.Buffer
	err := WriteOTel(&buf, probe.RunResult{
		EndedAt: time.Unix(10, 0),
		Summary: latency.Summary{Count: 1, P99: time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `"resourceMetrics"`) || !strings.Contains(got, `"p99.http.latency"`) {
		t.Fatalf("unexpected otel export: %s", got)
	}
}

func TestSpanMarkdownAndPrometheusExports(t *testing.T) {
	report := spans.Report{
		TraceCount:   1,
		SpanCount:    2,
		RequestCount: 1,
		Summary:      latency.Summary{Count: 1, P99: 200 * time.Millisecond},
		Routes: []spans.GroupSummary{{
			Name:    "/checkout",
			Count:   1,
			Summary: latency.Summary{Count: 1, P99: 200 * time.Millisecond},
		}},
		Dependencies: []spans.DependencySummary{{
			Name:    "redis",
			Type:    "peer",
			Count:   1,
			Total:   120 * time.Millisecond,
			Summary: latency.Summary{Count: 1, P95: 120 * time.Millisecond},
		}},
	}
	var md bytes.Buffer
	WriteSpanMarkdown(&md, report)
	if !strings.Contains(md.String(), "# p99 span report") || !strings.Contains(md.String(), "redis") {
		t.Fatalf("unexpected markdown: %s", md.String())
	}
	var prom bytes.Buffer
	WriteSpanPrometheus(&prom, report)
	if !strings.Contains(prom.String(), "p99_span_dependency_time_seconds_total") {
		t.Fatalf("unexpected prometheus: %s", prom.String())
	}
}
