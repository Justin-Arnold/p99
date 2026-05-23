package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/spans"
)

func WritePrometheus(w io.Writer, result probe.RunResult) {
	fmt.Fprintln(w, "# HELP p99_http_requests_total Total HTTP probe requests.")
	fmt.Fprintln(w, "# TYPE p99_http_requests_total counter")
	fmt.Fprintf(w, "p99_http_requests_total %d\n", result.Summary.Count)
	fmt.Fprintln(w, "# HELP p99_http_request_errors_total Total HTTP probe errors.")
	fmt.Fprintln(w, "# TYPE p99_http_request_errors_total counter")
	fmt.Fprintf(w, "p99_http_request_errors_total %d\n", result.Summary.Errors)
	fmt.Fprintln(w, "# HELP p99_http_error_rate HTTP probe error rate.")
	fmt.Fprintln(w, "# TYPE p99_http_error_rate gauge")
	fmt.Fprintf(w, "p99_http_error_rate %g\n", result.Summary.ErrorRate)
	writeLatencyQuantiles(w, "p99_http_latency_quantile_seconds", result.Summary)
	writeDistributionBuckets(w, "p99_http_latency_bucket_requests", result.Histogram)
	if len(result.Errors) > 0 {
		keys := make([]string, 0, len(result.Errors))
		for key := range result.Errors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintln(w, "# HELP p99_http_errors_by_class_total HTTP probe errors by class.")
		fmt.Fprintln(w, "# TYPE p99_http_errors_by_class_total counter")
		for _, key := range keys {
			fmt.Fprintf(w, "p99_http_errors_by_class_total{class=%q} %d\n", key, result.Errors[key])
		}
	}
}

func WriteSpanPrometheus(w io.Writer, report spans.Report) {
	fmt.Fprintln(w, "# HELP p99_span_requests_total Total request traces in the span input.")
	fmt.Fprintln(w, "# TYPE p99_span_requests_total counter")
	fmt.Fprintf(w, "p99_span_requests_total %d\n", report.RequestCount)
	fmt.Fprintln(w, "# HELP p99_span_request_errors_total Request traces marked as errors.")
	fmt.Fprintln(w, "# TYPE p99_span_request_errors_total counter")
	fmt.Fprintf(w, "p99_span_request_errors_total %d\n", report.Summary.Errors)
	writeLatencyQuantiles(w, "p99_span_request_latency_quantile_seconds", report.Summary)
	for _, route := range report.Routes {
		writeGroupQuantiles(w, "p99_span_route_latency_quantile_seconds", "route", route.Name, route.Summary)
	}
	for _, dep := range report.Dependencies {
		writeGroupQuantiles(w, "p99_span_dependency_latency_quantile_seconds", "dependency", dep.Name, dep.Summary)
		fmt.Fprintf(w, "p99_span_dependency_time_seconds_total{dependency=%q,type=%q} %g\n", dep.Name, dep.Type, dep.Total.Seconds())
	}
}

func writeLatencyQuantiles(w io.Writer, name string, s latency.Summary) {
	fmt.Fprintf(w, "# HELP %s Latency quantiles in seconds.\n", name)
	fmt.Fprintf(w, "# TYPE %s gauge\n", name)
	fmt.Fprintf(w, "%s{quantile=\"0.5\"} %g\n", name, s.P50.Seconds())
	fmt.Fprintf(w, "%s{quantile=\"0.9\"} %g\n", name, s.P90.Seconds())
	fmt.Fprintf(w, "%s{quantile=\"0.95\"} %g\n", name, s.P95.Seconds())
	fmt.Fprintf(w, "%s{quantile=\"0.99\"} %g\n", name, s.P99.Seconds())
	fmt.Fprintf(w, "%s{quantile=\"0.999\"} %g\n", name, s.P999.Seconds())
	fmt.Fprintf(w, "%s{quantile=\"1\"} %g\n", name, s.Max.Seconds())
}

func writeGroupQuantiles(w io.Writer, name, label, value string, s latency.Summary) {
	fmt.Fprintf(w, "%s{%s=%q,quantile=\"0.5\"} %g\n", name, label, value, s.P50.Seconds())
	fmt.Fprintf(w, "%s{%s=%q,quantile=\"0.95\"} %g\n", name, label, value, s.P95.Seconds())
	fmt.Fprintf(w, "%s{%s=%q,quantile=\"0.99\"} %g\n", name, label, value, s.P99.Seconds())
}

func writeDistributionBuckets(w io.Writer, name string, buckets []latency.Bucket) {
	if len(buckets) == 0 {
		return
	}
	fmt.Fprintf(w, "# HELP %s Cumulative latency distribution bucket counts by upper bound.\n", name)
	fmt.Fprintf(w, "# TYPE %s gauge\n", name)
	var cumulative int
	for _, bucket := range buckets {
		cumulative += bucket.Count
		le := "+Inf"
		if bucket.UpperBoundNS >= 0 {
			le = fmt.Sprintf("%g", float64(bucket.UpperBoundNS)/1e9)
		}
		fmt.Fprintf(w, "%s{le=%q} %d\n", name, le, cumulative)
	}
	if buckets[len(buckets)-1].UpperBoundNS >= 0 {
		fmt.Fprintf(w, "%s{le=%q} %d\n", name, "+Inf", cumulative)
	}
}
