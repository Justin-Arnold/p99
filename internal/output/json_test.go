package output

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
	"github.com/justin/p99/internal/runtimesignal"
)

func TestJSONRoundTripShape(t *testing.T) {
	var buf bytes.Buffer
	result := probe.RunResult{
		Version: probe.ResultVersion,
		Summary: latency.Summary{
			Count: 1,
			P99:   time.Millisecond,
		},
		Shape: latency.Shape{Kind: "stable"},
	}
	if err := WriteJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"version": 1`)) {
		t.Fatalf("unexpected json: %s", buf.String())
	}
}

func TestJSONFileFullRoundTrip(t *testing.T) {
	started := time.Unix(100, 1).UTC()
	ended := time.Unix(101, 2).UTC()
	result := probe.RunResult{
		Version: probe.ResultVersion,
		Config: probe.HTTPConfig{
			URL:             "https://example.test/search",
			Method:          "POST",
			Duration:        time.Second,
			RPS:             10,
			Concurrency:     2,
			Timeout:         500 * time.Millisecond,
			Headers:         map[string]string{"X-Test": "present"},
			RequestBodySize: 12,
			ExpectedStatus:  []int{201},
			SlowSamples:     3,
		},
		StartedAt: started,
		EndedAt:   ended,
		Summary: latency.Summary{
			Count:     3,
			Success:   2,
			Errors:    1,
			ErrorRate: 1.0 / 3.0,
			Min:       10 * time.Millisecond,
			P50:       20 * time.Millisecond,
			P90:       30 * time.Millisecond,
			P95:       30 * time.Millisecond,
			P99:       30 * time.Millisecond,
			P999:      30 * time.Millisecond,
			Max:       30 * time.Millisecond,
		},
		Histogram: []latency.Bucket{
			{UpperBoundNS: int64(10 * time.Millisecond), Count: 1},
			{UpperBoundNS: int64(30 * time.Millisecond), Count: 2},
		},
		SlowSamples: []latency.SlowSample{{
			Latency:   30 * time.Millisecond,
			Timestamp: started,
			Status:    500,
			Error:     "HTTP_5xx",
			Method:    "POST",
			URL:       "/search",
		}},
		Errors: map[string]int{"HTTP_5xx": 1},
		Shape:  latency.Shape{Kind: "spiky", Notes: []string{"tail moved"}, Hints: []string{"check retries"}},
		Runtime: &runtimesignal.Correlation{
			Before: runtimesignal.Snapshot{At: started, BaseURL: "http://localhost:8080", Goroutines: u64(10)},
			After:  runtimesignal.Snapshot{At: ended, BaseURL: "http://localhost:8080", Goroutines: u64(12)},
			Delta:  runtimesignal.Delta{Goroutines: i64(2)},
			Notes:  []string{"runtime moved"},
		},
	}

	path := filepath.Join(t.TempDir(), "run.json")
	if err := WriteJSONFile(path, result); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, result) {
		t.Fatalf("round trip mismatch\ngot:  %#v\nwant: %#v", got, result)
	}
}

func u64(v uint64) *uint64 {
	return &v
}

func i64(v int64) *int64 {
	return &v
}
