package compare

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

type Delta struct {
	Before     time.Duration
	After      time.Duration
	Change     time.Duration
	ChangeRate float64
}

type Result struct {
	P50          Delta
	P95          Delta
	P99          Delta
	P999         Delta
	Max          Delta
	ErrorRate    FloatDelta
	RequestCount IntDelta
}

type FloatDelta struct {
	Before     float64
	After      float64
	Change     float64
	ChangeRate float64
}

type IntDelta struct {
	Before     int
	After      int
	Change     int
	ChangeRate float64
}

func Compare(before, after probe.RunResult) Result {
	return Result{
		P50:          durationDelta(before.Summary.P50, after.Summary.P50),
		P95:          durationDelta(before.Summary.P95, after.Summary.P95),
		P99:          durationDelta(before.Summary.P99, after.Summary.P99),
		P999:         durationDelta(before.Summary.P999, after.Summary.P999),
		Max:          durationDelta(before.Summary.Max, after.Summary.Max),
		ErrorRate:    floatDelta(before.Summary.ErrorRate, after.Summary.ErrorRate),
		RequestCount: intDelta(before.Summary.Count, after.Summary.Count),
	}
}

func WriteText(w io.Writer, r Result) {
	fmt.Fprintln(w, "Metric    Before       After        Change")
	writeDuration(w, "p50", r.P50)
	writeDuration(w, "p95", r.P95)
	writeDuration(w, "p99", r.P99)
	writeDuration(w, "p999", r.P999)
	writeDuration(w, "max", r.Max)
	fmt.Fprintf(w, "%-8s  %-10.3f%%  %-10.3f%%  %+0.3f%% (%+.2f%%)\n", "errors", r.ErrorRate.Before*100, r.ErrorRate.After*100, r.ErrorRate.Change*100, r.ErrorRate.ChangeRate*100)
	fmt.Fprintf(w, "%-8s  %-10d  %-10d  %+d (%+.2f%%)\n", "count", r.RequestCount.Before, r.RequestCount.After, r.RequestCount.Change, r.RequestCount.ChangeRate*100)
}

func writeDuration(w io.Writer, name string, d Delta) {
	fmt.Fprintf(w, "%-8s  %-10s  %-10s  %+s (%+.2f%%)\n",
		name,
		timeutil.FormatDuration(d.Before),
		timeutil.FormatDuration(d.After),
		timeutil.FormatSignedDuration(d.Change),
		d.ChangeRate*100,
	)
}

func durationDelta(before, after time.Duration) Delta {
	return Delta{Before: before, After: after, Change: after - before, ChangeRate: changeRate(float64(before), float64(after))}
}

func floatDelta(before, after float64) FloatDelta {
	return FloatDelta{Before: before, After: after, Change: after - before, ChangeRate: changeRate(before, after)}
}

func intDelta(before, after int) IntDelta {
	return IntDelta{Before: before, After: after, Change: after - before, ChangeRate: changeRate(float64(before), float64(after))}
}

func changeRate(before, after float64) float64 {
	if before == 0 {
		if after == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return (after - before) / before
}
