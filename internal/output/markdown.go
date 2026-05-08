package output

import (
	"fmt"
	"io"

	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/spans"
	"github.com/justin/p99/internal/timeutil"
)

func WriteMarkdown(w io.Writer, result probe.RunResult) {
	s := result.Summary
	fmt.Fprintln(w, "# p99 report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Requests: %d\n", s.Count)
	fmt.Fprintf(w, "- Success: %d\n", s.Success)
	fmt.Fprintf(w, "- Errors: %d\n", s.Errors)
	fmt.Fprintf(w, "- Error rate: %.3f%%\n", s.ErrorRate*100)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Metric | Value |")
	fmt.Fprintln(w, "| --- | ---: |")
	fmt.Fprintf(w, "| min | %s |\n", timeutil.FormatDuration(s.Min))
	fmt.Fprintf(w, "| p50 | %s |\n", timeutil.FormatDuration(s.P50))
	fmt.Fprintf(w, "| p90 | %s |\n", timeutil.FormatDuration(s.P90))
	fmt.Fprintf(w, "| p95 | %s |\n", timeutil.FormatDuration(s.P95))
	fmt.Fprintf(w, "| p99 | %s |\n", timeutil.FormatDuration(s.P99))
	fmt.Fprintf(w, "| p999 | %s |\n", timeutil.FormatDuration(s.P999))
	fmt.Fprintf(w, "| max | %s |\n", timeutil.FormatDuration(s.Max))
	if result.Runtime != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Runtime Signals")
		fmt.Fprintln(w)
		WriteRuntime(w, *result.Runtime)
	}
}

func WriteSpanMarkdown(w io.Writer, report spans.Report) {
	fmt.Fprintln(w, "# p99 span report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Traces: %d\n", report.TraceCount)
	fmt.Fprintf(w, "- Spans: %d\n", report.SpanCount)
	fmt.Fprintf(w, "- Requests: %d\n", report.RequestCount)
	fmt.Fprintf(w, "- Error rate: %.3f%%\n", report.Summary.ErrorRate*100)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Request Latency")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Metric | Value |")
	fmt.Fprintln(w, "| --- | ---: |")
	fmt.Fprintf(w, "| p50 | %s |\n", timeutil.FormatDuration(report.Summary.P50))
	fmt.Fprintf(w, "| p95 | %s |\n", timeutil.FormatDuration(report.Summary.P95))
	fmt.Fprintf(w, "| p99 | %s |\n", timeutil.FormatDuration(report.Summary.P99))
	fmt.Fprintf(w, "| p999 | %s |\n", timeutil.FormatDuration(report.Summary.P999))
	fmt.Fprintf(w, "| max | %s |\n", timeutil.FormatDuration(report.Summary.Max))
	if len(report.Routes) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Routes")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Route | Count | Errors | p95 | p99 |")
		fmt.Fprintln(w, "| --- | ---: | ---: | ---: | ---: |")
		for _, route := range limitGroups(report.Routes, 10) {
			fmt.Fprintf(w, "| %s | %d | %d | %s | %s |\n", route.Name, route.Count, route.Summary.Errors, timeutil.FormatDuration(route.Summary.P95), timeutil.FormatDuration(route.Summary.P99))
		}
	}
	if len(report.Dependencies) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Dependencies")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Dependency | Type | Count | Total | p95 | Share |")
		fmt.Fprintln(w, "| --- | --- | ---: | ---: | ---: | ---: |")
		for _, dep := range limitDependencies(report.Dependencies, 10) {
			fmt.Fprintf(w, "| %s | %s | %d | %s | %s | %.1f%% |\n", dep.Name, dep.Type, dep.Count, timeutil.FormatDuration(dep.Total), timeutil.FormatDuration(dep.Summary.P95), dep.ShareOfRequests*100)
		}
	}
	if len(report.Hints) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Hints")
		fmt.Fprintln(w)
		for _, hint := range report.Hints {
			fmt.Fprintf(w, "- %s\n", hint)
		}
	}
}
