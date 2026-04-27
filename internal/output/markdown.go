package output

import (
	"fmt"
	"io"

	"github.com/justin/p99/internal/probe"
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
}
