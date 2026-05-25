package compare

import (
	"strings"
	"testing"
	"time"

	"github.com/Justin-Arnold/p99/internal/latency"
	"github.com/Justin-Arnold/p99/internal/probe"
)

func TestCompareMath(t *testing.T) {
	before := probe.RunResult{Summary: latency.Summary{Count: 100, P99: 100 * time.Millisecond, ErrorRate: 0.01}}
	after := probe.RunResult{Summary: latency.Summary{Count: 90, P99: 125 * time.Millisecond, ErrorRate: 0.02}}
	got := Compare(before, after)
	if got.P99.Change != 25*time.Millisecond {
		t.Fatalf("p99 change got %s", got.P99.Change)
	}
	if got.P99.ChangeRate != 0.25 {
		t.Fatalf("p99 change rate got %v", got.P99.ChangeRate)
	}
	if got.RequestCount.Change != -10 {
		t.Fatalf("count change got %d", got.RequestCount.Change)
	}
}

func TestEvaluateRunThresholds(t *testing.T) {
	result := probe.RunResult{Summary: latency.Summary{P99: 300 * time.Millisecond, ErrorRate: 0.01}}
	failures := EvaluateRun(result, RunThresholds{P99Under: 250 * time.Millisecond, ErrorRateUnder: 0.005})
	if len(failures) != 2 {
		t.Fatalf("failures got %#v", failures)
	}
}

func TestEvaluateCompareThresholds(t *testing.T) {
	result := Compare(
		probe.RunResult{Summary: latency.Summary{
			Count:     100,
			P50:       50 * time.Millisecond,
			P95:       100 * time.Millisecond,
			P99:       100 * time.Millisecond,
			P999:      150 * time.Millisecond,
			Max:       200 * time.Millisecond,
			ErrorRate: 0.01,
		}},
		probe.RunResult{Summary: latency.Summary{
			Count:     80,
			P50:       70 * time.Millisecond,
			P95:       140 * time.Millisecond,
			P99:       130 * time.Millisecond,
			P999:      220 * time.Millisecond,
			Max:       300 * time.Millisecond,
			ErrorRate: 0.03,
		}},
	)
	failures := EvaluateCompare(result, CompareThresholds{
		MaxP50Regression:       0.20,
		MaxP95Regression:       0.20,
		MaxP99Regression:       0.20,
		MaxP999Regression:      0.20,
		MaxMaxRegression:       0.20,
		MaxErrorRateRegression: 0.50,
		MaxErrorRate:           0.02,
		MinRequestCount:        90,
		MaxRequestDrop:         0.10,
		P95Under:               120 * time.Millisecond,
		P999Under:              200 * time.Millisecond,
		MaxUnder:               250 * time.Millisecond,
	})
	joined := strings.Join(failures, "\n")
	for _, want := range []string{
		"p50 regression",
		"p95 regression",
		"p99 regression",
		"p999 regression",
		"max regression",
		"error rate regression",
		"error rate",
		"request count",
		"request count drop",
		"p95",
		"p999",
		"max",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in failures: %#v", want, failures)
		}
	}
}

func TestEvaluateCompareThresholdsIgnoresImprovements(t *testing.T) {
	result := Compare(
		probe.RunResult{Summary: latency.Summary{Count: 100, P99: 100 * time.Millisecond, ErrorRate: 0.02}},
		probe.RunResult{Summary: latency.Summary{Count: 110, P99: 90 * time.Millisecond, ErrorRate: 0.01}},
	)
	failures := EvaluateCompare(result, CompareThresholds{MaxP99Regression: 0.01, MaxErrorRateRegression: 0.01, MaxRequestDrop: 0.01})
	if len(failures) != 0 {
		t.Fatalf("failures got %#v", failures)
	}
}
