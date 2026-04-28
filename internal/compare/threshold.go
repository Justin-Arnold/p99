package compare

import (
	"fmt"
	"time"

	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/timeutil"
)

type RunThresholds struct {
	P99Under       time.Duration
	ErrorRateUnder float64
}

type CompareThresholds struct {
	MaxP99Regression float64
}

func EvaluateRun(result probe.RunResult, t RunThresholds) []string {
	var failures []string
	if t.P99Under > 0 && result.Summary.P99 > t.P99Under {
		failures = append(failures, fmt.Sprintf("p99 %s exceeded budget %s", timeutil.FormatDuration(result.Summary.P99), timeutil.FormatDuration(t.P99Under)))
	}
	if t.ErrorRateUnder > 0 && result.Summary.ErrorRate > t.ErrorRateUnder {
		failures = append(failures, fmt.Sprintf("error rate %.3f%% exceeded budget %.3f%%", result.Summary.ErrorRate*100, t.ErrorRateUnder*100))
	}
	return failures
}

func EvaluateCompare(result Result, t CompareThresholds) []string {
	var failures []string
	if t.MaxP99Regression > 0 && result.P99.ChangeRate > t.MaxP99Regression {
		failures = append(failures, fmt.Sprintf("p99 regression %.2f%% exceeded budget %.2f%%", result.P99.ChangeRate*100, t.MaxP99Regression*100))
	}
	return failures
}
