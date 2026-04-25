package latency

import (
	"testing"
	"time"
)

func TestAnalyzeShapeInsufficient(t *testing.T) {
	if got := AnalyzeShape(nil); got.Kind != "insufficient_signal" {
		t.Fatalf("got %q", got.Kind)
	}
}

func TestAnalyzeShapeStable(t *testing.T) {
	points := make([]TimedLatency, 50)
	now := time.Now()
	for i := range points {
		points[i] = TimedLatency{At: now.Add(time.Duration(i) * time.Millisecond), Latency: 20*time.Millisecond + time.Duration(i%3)*time.Millisecond}
	}
	if got := AnalyzeShape(points); got.Kind != "stable" {
		t.Fatalf("got %q", got.Kind)
	}
}

func TestAnalyzeShapeSpiky(t *testing.T) {
	points := make([]TimedLatency, 100)
	now := time.Now()
	for i := range points {
		lat := 20 * time.Millisecond
		if i >= 98 {
			lat = 400 * time.Millisecond
		}
		points[i] = TimedLatency{At: now.Add(time.Duration(i) * time.Millisecond), Latency: lat}
	}
	if got := AnalyzeShape(points); got.Kind != "spiky" {
		t.Fatalf("got %q", got.Kind)
	}
}

func TestAnalyzeShapeDegrading(t *testing.T) {
	points := make([]TimedLatency, 60)
	now := time.Now()
	for i := range points {
		lat := 20 * time.Millisecond
		if i >= 40 {
			lat = 100 * time.Millisecond
		}
		points[i] = TimedLatency{At: now.Add(time.Duration(i) * time.Millisecond), Latency: lat}
	}
	if got := AnalyzeShape(points); got.Kind != "degrading_over_time" {
		t.Fatalf("got %q", got.Kind)
	}
}
