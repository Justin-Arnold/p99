package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return 0, fmt.Errorf("duration is required")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("duration must be non-negative")
	}
	return d, nil
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "-" + FormatDuration(-d)
	}
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.3fs", float64(d)/float64(time.Second))
	case d >= time.Millisecond:
		return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.3fus", float64(d)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}

func FormatSignedDuration(d time.Duration) string {
	if d < 0 {
		return FormatDuration(d)
	}
	return "+" + FormatDuration(d)
}

func ParsePercentThreshold(s string) (float64, error) {
	if strings.TrimSpace(s) == "" {
		return 0, fmt.Errorf("percent threshold is required")
	}
	raw := strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("percent threshold must be non-negative")
	}
	return v / 100, nil
}
