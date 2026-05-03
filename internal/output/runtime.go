package output

import (
	"fmt"
	"io"
	"time"

	"github.com/justin/p99/internal/runtimesignal"
	"github.com/justin/p99/internal/timeutil"
)

func WriteRuntime(w io.Writer, c runtimesignal.Correlation) {
	fmt.Fprintln(w, "Runtime signals:")
	writeIntDelta(w, "goroutines", c.Before.Goroutines, c.After.Goroutines, c.Delta.Goroutines)
	writeBytesDelta(w, "heap alloc", c.Before.HeapAllocBytes, c.After.HeapAllocBytes, c.Delta.HeapAllocBytes)
	writeBytesDelta(w, "heap in use", c.Before.HeapInUseBytes, c.After.HeapInUseBytes, c.Delta.HeapInUseBytes)
	writeUintDelta(w, "GC cycles", c.Before.GCCycles, c.After.GCCycles, c.Delta.GCCycles)
	writeDurationDelta(w, "GC pause total", c.Before.GCPauseTotalNS, c.After.GCPauseTotalNS, c.Delta.GCPauseTotalNS)
	writeIntDelta(w, "mutex samples", c.Before.MutexSamples, c.After.MutexSamples, c.Delta.MutexSamples)
	writeIntDelta(w, "block samples", c.Before.BlockSamples, c.After.BlockSamples, c.Delta.BlockSamples)
	if c.Delta.SchedulerPauses != nil {
		fmt.Fprintf(w, "  scheduler pauses: +%d events, %s total\n", c.Delta.SchedulerPauses.Count, timeutil.FormatDuration(ns(c.Delta.SchedulerPauses.SumNS)))
	}
	if c.Delta.DBPoolWait != nil {
		fmt.Fprintf(w, "  DB pool wait: +%d waits, %s total\n", c.Delta.DBPoolWait.Count, timeutil.FormatDuration(ns(c.Delta.DBPoolWait.DurationNS)))
	}
	for _, note := range c.Notes {
		fmt.Fprintf(w, "  Hint: %s\n", note)
	}
	for _, warning := range append(c.Before.Warnings, c.After.Warnings...) {
		fmt.Fprintf(w, "  Warning: %s\n", warning)
	}
}

func writeIntDelta(w io.Writer, label string, before, after *uint64, delta *int64) {
	if before == nil || after == nil || delta == nil {
		fmt.Fprintf(w, "  %s: unavailable\n", label)
		return
	}
	fmt.Fprintf(w, "  %s: %d -> %d (%+d)\n", label, *before, *after, *delta)
}

func writeUintDelta(w io.Writer, label string, before, after, delta *uint64) {
	if before == nil || after == nil || delta == nil {
		fmt.Fprintf(w, "  %s: unavailable\n", label)
		return
	}
	fmt.Fprintf(w, "  %s: %d -> %d (+%d)\n", label, *before, *after, *delta)
}

func writeBytesDelta(w io.Writer, label string, before, after *uint64, delta *int64) {
	if before == nil || after == nil || delta == nil {
		fmt.Fprintf(w, "  %s: unavailable\n", label)
		return
	}
	fmt.Fprintf(w, "  %s: %s -> %s (%s)\n", label, formatBytes(*before), formatBytes(*after), formatSignedBytes(*delta))
}

func writeDurationDelta(w io.Writer, label string, before, after, delta *uint64) {
	if before == nil || after == nil || delta == nil {
		fmt.Fprintf(w, "  %s: unavailable\n", label)
		return
	}
	fmt.Fprintf(w, "  %s: %s -> %s (+%s)\n", label, timeutil.FormatDuration(ns(*before)), timeutil.FormatDuration(ns(*after)), timeutil.FormatDuration(ns(*delta)))
}

func ns(v uint64) time.Duration {
	return time.Duration(v)
}

func formatBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%dB", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(v)/float64(div), "KMGTPE"[exp])
}

func formatSignedBytes(v int64) string {
	if v < 0 {
		return "-" + formatBytes(uint64(-v))
	}
	return "+" + formatBytes(uint64(v))
}
