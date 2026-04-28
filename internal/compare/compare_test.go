package compare

import (
	"strings"
	"testing"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
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
		probe.RunResult{Summary: latency.Summary{P99: 100 * time.Millisecond}},
		probe.RunResult{Summary: latency.Summary{P99: 130 * time.Millisecond}},
	)
	failures := EvaluateCompare(result, CompareThresholds{MaxP99Regression: 0.20})
	if len(failures) != 1 || !strings.Contains(failures[0], "p99 regression") {
		t.Fatalf("failures got %#v", failures)
	}
}
