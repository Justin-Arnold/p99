package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/justin/p99/internal/latency"
	"github.com/justin/p99/internal/probe"
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
