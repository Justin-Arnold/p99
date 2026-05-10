package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	p99compare "github.com/justin/p99/internal/compare"
	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/timeutil"
)

func runCompare(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	var maxP50RegressionText, maxP95RegressionText, maxP99RegressionText, maxP999RegressionText, maxMaxRegressionText string
	var maxErrorRateRegressionText, maxErrorRateText, maxRequestDropText string
	var p50UnderText, p95UnderText, p99UnderText, p999UnderText, maxUnderText string
	var minRequestCount int
	fs.StringVar(&maxP50RegressionText, "max-p50-regression", "", "fail if p50 regression exceeds percent")
	fs.StringVar(&maxP95RegressionText, "max-p95-regression", "", "fail if p95 regression exceeds percent")
	fs.StringVar(&maxP99RegressionText, "max-p99-regression", "", "fail if p99 regression exceeds percent")
	fs.StringVar(&maxP999RegressionText, "max-p999-regression", "", "fail if p999 regression exceeds percent")
	fs.StringVar(&maxMaxRegressionText, "max-max-regression", "", "fail if max latency regression exceeds percent")
	fs.StringVar(&maxErrorRateRegressionText, "max-error-rate-regression", "", "fail if error rate regression exceeds percent")
	fs.StringVar(&maxErrorRateText, "error-rate-under", "", "fail if after error rate exceeds percent")
	fs.StringVar(&maxErrorRateText, "max-error-rate", "", "fail if after error rate exceeds percent")
	fs.IntVar(&minRequestCount, "min-request-count", 0, "fail if after request count is below this value")
	fs.StringVar(&maxRequestDropText, "max-request-drop", "", "fail if request count drop exceeds percent")
	fs.StringVar(&p50UnderText, "p50-under", "", "fail if after p50 is above duration")
	fs.StringVar(&p95UnderText, "p95-under", "", "fail if after p95 is above duration")
	fs.StringVar(&p99UnderText, "p99-under", "", "fail if after p99 is above duration")
	fs.StringVar(&p999UnderText, "p999-under", "", "fail if after p999 is above duration")
	fs.StringVar(&maxUnderText, "max-under", "", "fail if after max latency is above duration")
	valueFlags := map[string]bool{
		"max-p50-regression": true, "max-p95-regression": true, "max-p99-regression": true,
		"max-p999-regression": true, "max-max-regression": true, "max-error-rate-regression": true,
		"error-rate-under": true, "max-error-rate": true, "min-request-count": true, "max-request-drop": true,
		"p50-under": true, "p95-under": true, "p99-under": true, "p999-under": true, "max-under": true,
	}
	if err := parse(fs, args, valueFlags); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "compare requires before and after run JSON files")
		return 2
	}
	before, err := output.ReadJSON(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "read before: %v\n", err)
		return 1
	}
	after, err := output.ReadJSON(fs.Arg(1))
	if err != nil {
		fmt.Fprintf(stderr, "read after: %v\n", err)
		return 1
	}
	result := p99compare.Compare(before, after)
	p99compare.WriteText(stdout, result)

	thresholds, err := parseCompareThresholds(compareThresholdTexts{
		maxP50Regression:       maxP50RegressionText,
		maxP95Regression:       maxP95RegressionText,
		maxP99Regression:       maxP99RegressionText,
		maxP999Regression:      maxP999RegressionText,
		maxMaxRegression:       maxMaxRegressionText,
		maxErrorRateRegression: maxErrorRateRegressionText,
		maxErrorRate:           maxErrorRateText,
		maxRequestDrop:         maxRequestDropText,
		p50Under:               p50UnderText,
		p95Under:               p95UnderText,
		p99Under:               p99UnderText,
		p999Under:              p999UnderText,
		maxUnder:               maxUnderText,
		minRequestCount:        minRequestCount,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	failures := p99compare.EvaluateCompare(result, thresholds)
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintf(stderr, "threshold failed: %s\n", failure)
		}
		return 1
	}
	return 0
}

type compareThresholdTexts struct {
	maxP50Regression       string
	maxP95Regression       string
	maxP99Regression       string
	maxP999Regression      string
	maxMaxRegression       string
	maxErrorRateRegression string
	maxErrorRate           string
	maxRequestDrop         string
	p50Under               string
	p95Under               string
	p99Under               string
	p999Under              string
	maxUnder               string
	minRequestCount        int
}

func parseCompareThresholds(text compareThresholdTexts) (p99compare.CompareThresholds, error) {
	var t p99compare.CompareThresholds
	var err error
	t.MinRequestCount = text.minRequestCount
	if t.MaxP50Regression, err = parseOptionalPercent("max-p50-regression", text.maxP50Regression); err != nil {
		return t, err
	}
	if t.MaxP95Regression, err = parseOptionalPercent("max-p95-regression", text.maxP95Regression); err != nil {
		return t, err
	}
	if t.MaxP99Regression, err = parseOptionalPercent("max-p99-regression", text.maxP99Regression); err != nil {
		return t, err
	}
	if t.MaxP999Regression, err = parseOptionalPercent("max-p999-regression", text.maxP999Regression); err != nil {
		return t, err
	}
	if t.MaxMaxRegression, err = parseOptionalPercent("max-max-regression", text.maxMaxRegression); err != nil {
		return t, err
	}
	if t.MaxErrorRateRegression, err = parseOptionalPercent("max-error-rate-regression", text.maxErrorRateRegression); err != nil {
		return t, err
	}
	if t.MaxErrorRate, err = parseOptionalPercent("error-rate-under", text.maxErrorRate); err != nil {
		return t, err
	}
	if t.MaxRequestDrop, err = parseOptionalPercent("max-request-drop", text.maxRequestDrop); err != nil {
		return t, err
	}
	if t.P50Under, err = parseOptionalDuration("p50-under", text.p50Under); err != nil {
		return t, err
	}
	if t.P95Under, err = parseOptionalDuration("p95-under", text.p95Under); err != nil {
		return t, err
	}
	if t.P99Under, err = parseOptionalDuration("p99-under", text.p99Under); err != nil {
		return t, err
	}
	if t.P999Under, err = parseOptionalDuration("p999-under", text.p999Under); err != nil {
		return t, err
	}
	if t.MaxUnder, err = parseOptionalDuration("max-under", text.maxUnder); err != nil {
		return t, err
	}
	return t, nil
}

func parseOptionalPercent(name, raw string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := timeutil.ParsePercentThreshold(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}

func parseOptionalDuration(name, raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := timeutil.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}
