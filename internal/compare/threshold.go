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
	MaxP50Regression       float64
	MaxP95Regression       float64
	MaxP99Regression       float64
	MaxP999Regression      float64
	MaxMaxRegression       float64
	MaxErrorRateRegression float64
	MaxErrorRate           float64
	MinRequestCount        int
	MaxRequestDrop         float64
	P50Under               time.Duration
	P95Under               time.Duration
	P99Under               time.Duration
	P999Under              time.Duration
	MaxUnder               time.Duration
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
	failures = append(failures, evaluateRegression("p50", result.P50.ChangeRate, t.MaxP50Regression)...)
	failures = append(failures, evaluateRegression("p95", result.P95.ChangeRate, t.MaxP95Regression)...)
	failures = append(failures, evaluateRegression("p99", result.P99.ChangeRate, t.MaxP99Regression)...)
	failures = append(failures, evaluateRegression("p999", result.P999.ChangeRate, t.MaxP999Regression)...)
	failures = append(failures, evaluateRegression("max", result.Max.ChangeRate, t.MaxMaxRegression)...)
	failures = append(failures, evaluateRegression("error rate", result.ErrorRate.ChangeRate, t.MaxErrorRateRegression)...)
	failures = append(failures, evaluateDurationBudget("p50", result.P50.After, t.P50Under)...)
	failures = append(failures, evaluateDurationBudget("p95", result.P95.After, t.P95Under)...)
	failures = append(failures, evaluateDurationBudget("p99", result.P99.After, t.P99Under)...)
	failures = append(failures, evaluateDurationBudget("p999", result.P999.After, t.P999Under)...)
	failures = append(failures, evaluateDurationBudget("max", result.Max.After, t.MaxUnder)...)
	if t.MaxErrorRate > 0 && result.ErrorRate.After > t.MaxErrorRate {
		failures = append(failures, fmt.Sprintf("error rate %.3f%% exceeded budget %.3f%%", result.ErrorRate.After*100, t.MaxErrorRate*100))
	}
	if t.MinRequestCount > 0 && result.RequestCount.After < t.MinRequestCount {
		failures = append(failures, fmt.Sprintf("request count %d was below minimum %d", result.RequestCount.After, t.MinRequestCount))
	}
	if t.MaxRequestDrop > 0 && -result.RequestCount.ChangeRate > t.MaxRequestDrop {
		failures = append(failures, fmt.Sprintf("request count drop %.2f%% exceeded budget %.2f%%", -result.RequestCount.ChangeRate*100, t.MaxRequestDrop*100))
	}
	return failures
}

func evaluateRegression(name string, actual, budget float64) []string {
	if budget <= 0 || actual <= budget {
		return nil
	}
	return []string{fmt.Sprintf("%s regression %.2f%% exceeded budget %.2f%%", name, actual*100, budget*100)}
}

func evaluateDurationBudget(name string, actual, budget time.Duration) []string {
	if budget <= 0 || actual <= budget {
		return nil
	}
	return []string{fmt.Sprintf("%s %s exceeded budget %s", name, timeutil.FormatDuration(actual), timeutil.FormatDuration(budget))}
}
