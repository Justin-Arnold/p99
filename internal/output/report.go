package output

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/Justin-Arnold/p99/internal/latency"
	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

type ReportOptions struct {
	Details     bool
	Histogram   bool
	SlowSamples bool
	Shape       bool
	Runtime     bool
}

func DetailedReportOptions() ReportOptions {
	return ReportOptions{
		Details:     true,
		Histogram:   true,
		SlowSamples: true,
		Shape:       true,
		Runtime:     true,
	}
}

func WriteReport(w io.Writer, result probe.RunResult, opts ReportOptions) {
	WriteSummary(w, result)
	if opts.Details || opts.Histogram {
		writeHistogram(w, result.Histogram)
	}
	if opts.Details || opts.SlowSamples {
		writeSlowSamples(w, result.SlowSamples)
	}
	if opts.Details || opts.Shape {
		writeShapeDetail(w, result)
	}
	if (opts.Details || opts.Runtime) && result.Runtime != nil {
		WriteRuntime(w, *result.Runtime)
	}
}

func writeHistogram(w io.Writer, buckets []latency.Bucket) {
	if len(buckets) == 0 {
		fmt.Fprintln(w, "Histogram: unavailable")
		return
	}
	fmt.Fprintln(w, "Histogram:")
	var cumulative int
	total := 0
	for _, bucket := range buckets {
		total += bucket.Count
	}
	for _, bucket := range buckets {
		cumulative += bucket.Count
		label := "+Inf"
		if bucket.UpperBoundNS >= 0 {
			label = "<= " + timeutil.FormatDuration(time.Duration(bucket.UpperBoundNS))
		}
		var pct float64
		if total > 0 {
			pct = float64(cumulative) / float64(total) * 100
		}
		fmt.Fprintf(w, "  %-12s count=%d cumulative=%d (%.1f%%)\n", label, bucket.Count, cumulative, pct)
	}
}

func writeSlowSamples(w io.Writer, samples []latency.SlowSample) {
	if len(samples) == 0 {
		fmt.Fprintln(w, "Slow samples: none")
		return
	}
	fmt.Fprintln(w, "Slow samples:")
	for _, sample := range samples {
		status := "-"
		if sample.Status != 0 {
			status = fmt.Sprint(sample.Status)
		}
		errClass := sample.Error
		if errClass == "" {
			errClass = "-"
		}
		fmt.Fprintf(w, "  %s  %s  status=%s  error=%s  %s %s\n",
			sample.Timestamp.Format(time.RFC3339),
			timeutil.FormatDuration(sample.Latency),
			status,
			errClass,
			sampleLabel(sample),
			sample.URL,
		)
	}
}

func sampleLabel(sample latency.SlowSample) string {
	if sample.Request == "" {
		return sample.Method
	}
	return sample.Request + " " + sample.Method
}

func writeShapeDetail(w io.Writer, result probe.RunResult) {
	fmt.Fprintln(w, "Shape analysis:")
	fmt.Fprintf(w, "  Kind: %s\n", result.Shape.Kind)
	for _, note := range result.Shape.Notes {
		fmt.Fprintf(w, "  Note: %s\n", note)
	}
	for _, hint := range result.Shape.Hints {
		fmt.Fprintf(w, "  Hint: %s\n", hint)
	}
	for _, explanation := range shapeExplanations(result.Shape.Kind) {
		fmt.Fprintf(w, "  Detail: %s\n", explanation)
	}
}

func shapeExplanations(kind string) []string {
	switch kind {
	case "stable":
		return []string{"The upper tail remains within a moderate multiple of the median. Still compare by route or dependency before generalizing."}
	case "spiky":
		return []string{"Most requests are close to the median, but a small number are much slower. Intermittent stalls, retries, lock contention, or noisy dependencies are common next checks."}
	case "bimodal":
		return []string{"Requests appear to land in separated latency bands. This often means different code paths or dependency states are mixed into one distribution."}
	case "degrading_over_time":
		return []string{"Later requests were slower than earlier requests. Queue buildup, resource exhaustion, GC pressure, and downstream slowdown are worth checking."}
	case "insufficient_signal":
		return []string{"The run did not produce enough evidence for a specific shape. Increase duration or segment by route/dependency if the result matters."}
	default:
		return []string{"No detailed explanation is available for this shape kind."}
	}
}

func sortedErrorKeys(errors map[string]int) []string {
	keys := make([]string, 0, len(errors))
	for key := range errors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
