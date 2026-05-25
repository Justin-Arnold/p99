package output

import (
	"fmt"
	"io"

	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

func WriteSummary(w io.Writer, result probe.RunResult) {
	s := result.Summary
	fmt.Fprintf(w, "Requests: %d (%d success, %d errors)\n", s.Count, s.Success, s.Errors)
	fmt.Fprintf(w, "Error rate: %.3f%%\n", s.ErrorRate*100)
	fmt.Fprintf(w, "Latency: min %s  p50 %s  p90 %s  p95 %s  p99 %s  p999 %s  max %s\n",
		timeutil.FormatDuration(s.Min),
		timeutil.FormatDuration(s.P50),
		timeutil.FormatDuration(s.P90),
		timeutil.FormatDuration(s.P95),
		timeutil.FormatDuration(s.P99),
		timeutil.FormatDuration(s.P999),
		timeutil.FormatDuration(s.Max),
	)
	if len(result.Errors) > 0 {
		fmt.Fprintln(w, "Errors:")
		for _, k := range sortedErrorKeys(result.Errors) {
			fmt.Fprintf(w, "  %s: %d\n", k, result.Errors[k])
		}
	}
	fmt.Fprintf(w, "Shape: %s\n", result.Shape.Kind)
	for _, note := range result.Shape.Notes {
		fmt.Fprintf(w, "  %s\n", note)
	}
	for _, hint := range result.Shape.Hints {
		fmt.Fprintf(w, "  Hint: %s\n", hint)
	}
	if result.Runtime != nil {
		WriteRuntime(w, *result.Runtime)
	}
}
