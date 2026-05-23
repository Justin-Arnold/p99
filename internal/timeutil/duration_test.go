package timeutil

import (
	"testing"
	"time"
)

func TestParseDurationRejectsNegative(t *testing.T) {
	if _, err := ParseDuration("-1s"); err == nil {
		t.Fatal("expected negative duration to fail")
	}
}

func TestParsePercentThreshold(t *testing.T) {
	got, err := ParsePercentThreshold("0.5")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0.005 {
		t.Fatalf("got %v, want 0.005", got)
	}

	got, err = ParsePercentThreshold("20%")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0.20 {
		t.Fatalf("got %v, want 0.20", got)
	}
}

func TestParsePercentThresholdRejectsNonFinite(t *testing.T) {
	for _, input := range []string{"NaN", "+Inf", "-Inf"} {
		if _, err := ParsePercentThreshold(input); err == nil {
			t.Fatalf("expected %q to fail", input)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(1500 * time.Microsecond); got != "1.500ms" {
		t.Fatalf("got %q", got)
	}
	if got := FormatDuration(-1500 * time.Microsecond); got != "-1.500ms" {
		t.Fatalf("got %q", got)
	}
	if got := FormatSignedDuration(1500 * time.Microsecond); got != "+1.500ms" {
		t.Fatalf("got %q", got)
	}
}
