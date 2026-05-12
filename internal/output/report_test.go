package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
)

func TestWriteReportDetails(t *testing.T) {
	result := probe.RunResult{
		Summary: latency.Summary{Count: 2, Success: 1, Errors: 1, P99: 200 * time.Millisecond},
		Histogram: []latency.Bucket{
			{UpperBoundNS: int64(100 * time.Millisecond), Count: 1},
			{UpperBoundNS: -1, Count: 1},
		},
		SlowSamples: []latency.SlowSample{{
			Latency:   200 * time.Millisecond,
			Timestamp: time.Unix(10, 0),
			Status:    500,
			Error:     "HTTP_5xx",
			Method:    "GET",
			URL:       "/slow",
		}},
		Shape: latency.Shape{Kind: "spiky"},
	}
	var buf bytes.Buffer
	WriteReport(&buf, result, DetailedReportOptions())
	got := buf.String()
	for _, want := range []string{"Histogram:", "Slow samples:", "Shape analysis:", "Detail:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
}
