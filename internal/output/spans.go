package output

import (
	"fmt"
	"io"

	"github.com/justin/p99/internal/spans"
	"github.com/justin/p99/internal/timeutil"
)

func WriteSpanReport(w io.Writer, report spans.Report) {
	fmt.Fprintf(w, "Traces: %d  Spans: %d  Requests: %d\n", report.TraceCount, report.SpanCount, report.RequestCount)
	s := report.Summary
	fmt.Fprintf(w, "Errors: %d (%.3f%%)\n", s.Errors, s.ErrorRate*100)
	fmt.Fprintf(w, "Request latency: min %s  p50 %s  p95 %s  p99 %s  p999 %s  max %s\n",
		timeutil.FormatDuration(s.Min),
		timeutil.FormatDuration(s.P50),
		timeutil.FormatDuration(s.P95),
		timeutil.FormatDuration(s.P99),
		timeutil.FormatDuration(s.P999),
		timeutil.FormatDuration(s.Max),
	)
	if len(report.Routes) > 0 {
		fmt.Fprintln(w, "Routes:")
		for _, route := range limitGroups(report.Routes, 10) {
			fmt.Fprintf(w, "  %s  count=%d  errors=%d  p50=%s  p95=%s  p99=%s\n",
				route.Name,
				route.Count,
				route.Summary.Errors,
				timeutil.FormatDuration(route.Summary.P50),
				timeutil.FormatDuration(route.Summary.P95),
				timeutil.FormatDuration(route.Summary.P99),
			)
		}
	}
	if len(report.Dependencies) > 0 {
		fmt.Fprintln(w, "Dependencies:")
		for _, dep := range limitDependencies(report.Dependencies, 10) {
			fmt.Fprintf(w, "  %s  type=%s  count=%d  total=%s  p95=%s  share=%.1f%%\n",
				dep.Name,
				dep.Type,
				dep.Count,
				timeutil.FormatDuration(dep.Total),
				timeutil.FormatDuration(dep.Summary.P95),
				dep.ShareOfRequests*100,
			)
		}
	}
	if len(report.SlowTraces) > 0 {
		fmt.Fprintln(w, "Slow traces:")
		for _, trace := range report.SlowTraces {
			fmt.Fprintf(w, "  %s  route=%s  latency=%s  dependency_time=%s\n",
				trace.TraceID,
				trace.Route,
				timeutil.FormatDuration(trace.Latency),
				timeutil.FormatDuration(trace.DependencyTime),
			)
		}
	}
	for _, hint := range report.Hints {
		fmt.Fprintf(w, "Hint: %s\n", hint)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
}

func limitGroups(groups []spans.GroupSummary, limit int) []spans.GroupSummary {
	if len(groups) <= limit {
		return groups
	}
	return groups[:limit]
}

func limitDependencies(deps []spans.DependencySummary, limit int) []spans.DependencySummary {
	if len(deps) <= limit {
		return deps
	}
	return deps[:limit]
}
