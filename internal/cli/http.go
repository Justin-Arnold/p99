package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Justin-Arnold/p99/internal/compare"
	"github.com/Justin-Arnold/p99/internal/output"
	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/runtimesignal"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

type headerFlags map[string]string

func (h headerFlags) String() string { return fmt.Sprint(map[string]string(h)) }

func (h headerFlags) Set(v string) error {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return fmt.Errorf("header must use Name: value")
	}
	h[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	return nil
}

type statusFlags []int

func (s *statusFlags) String() string { return fmt.Sprint([]int(*s)) }

func (s *statusFlags) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		code, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return err
		}
		if code < 100 || code > 599 {
			return fmt.Errorf("invalid status code %d", code)
		}
		*s = append(*s, code)
	}
	return nil
}

func runHTTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	var durationText, warmupText, timeoutText, outputPath, bodyFile, p99UnderText, errorRateUnderText string
	var runtimeURL, runtimeTimeoutText string
	var markdownOutput, prometheusOutput, otelOutput string
	headers := headerFlags{}
	statuses := statusFlags{}
	cfg := probe.HTTPConfig{Method: http.MethodGet, Duration: 10 * time.Second, RPS: 1, Concurrency: 1, Timeout: 10 * time.Second, SlowSamples: 10}
	fs.StringVar(&durationText, "duration", "10s", "measured duration")
	fs.Float64Var(&cfg.RPS, "rps", 1, "target requests per second")
	fs.IntVar(&cfg.Concurrency, "concurrency", 1, "maximum concurrent requests")
	fs.StringVar(&warmupText, "warmup", "0s", "warmup duration")
	fs.StringVar(&timeoutText, "timeout", "10s", "request timeout")
	fs.StringVar(&cfg.Method, "method", http.MethodGet, "HTTP method")
	fs.Var(headers, "H", "request header, Name: value")
	fs.Var(headers, "header", "request header, Name: value")
	fs.StringVar(&bodyFile, "body-file", "", "file to use as request body")
	fs.Var(&statuses, "status", "expected status code; may be repeated or comma-separated")
	fs.StringVar(&outputPath, "output", "", "write run JSON to path")
	fs.StringVar(&outputPath, "out", "", "write run JSON to path")
	fs.IntVar(&cfg.SlowSamples, "slow-samples", 10, "number of slow request samples to retain")
	fs.StringVar(&p99UnderText, "p99-under", "", "fail if p99 is above duration")
	fs.StringVar(&errorRateUnderText, "error-rate-under", "", "fail if error rate is above percent, e.g. 0.5")
	fs.StringVar(&runtimeURL, "runtime", "", "Go runtime base URL for correlation, e.g. http://localhost:8080")
	fs.StringVar(&runtimeTimeoutText, "runtime-timeout", "5s", "runtime endpoint timeout")
	fs.StringVar(&markdownOutput, "markdown-output", "", "write Markdown report to path")
	fs.StringVar(&prometheusOutput, "prometheus-output", "", "write Prometheus text metrics to path")
	fs.StringVar(&otelOutput, "otel-output", "", "write OpenTelemetry metrics JSON to path")

	valueFlags := map[string]bool{
		"duration": true, "rps": true, "concurrency": true, "warmup": true, "timeout": true,
		"method": true, "H": true, "header": true, "body-file": true, "status": true,
		"output": true, "out": true, "slow-samples": true, "p99-under": true, "error-rate-under": true,
		"runtime": true, "runtime-timeout": true,
		"markdown-output": true, "prometheus-output": true, "otel-output": true,
	}
	if err := parse(fs, args, valueFlags); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "http requires exactly one URL")
		return 2
	}

	var err error
	cfg.Duration, err = timeutil.ParseDuration(durationText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid duration: %v\n", err)
		return 2
	}
	cfg.Warmup, err = timeutil.ParseDuration(warmupText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid warmup: %v\n", err)
		return 2
	}
	cfg.Timeout, err = timeutil.ParseDuration(timeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid timeout: %v\n", err)
		return 2
	}
	if err := requirePositiveDuration("duration", cfg.Duration); err != nil {
		fmt.Fprintf(stderr, "invalid duration: %v\n", err)
		return 2
	}
	if err := requirePositiveFloat("rps", cfg.RPS); err != nil {
		fmt.Fprintf(stderr, "invalid rps: %v\n", err)
		return 2
	}
	if err := requirePositiveInt("concurrency", cfg.Concurrency); err != nil {
		fmt.Fprintf(stderr, "invalid concurrency: %v\n", err)
		return 2
	}
	if err := requirePositiveDuration("timeout", cfg.Timeout); err != nil {
		fmt.Fprintf(stderr, "invalid timeout: %v\n", err)
		return 2
	}
	if err := requireNonNegativeInt("slow-samples", cfg.SlowSamples); err != nil {
		fmt.Fprintf(stderr, "invalid slow-samples: %v\n", err)
		return 2
	}
	cfg.URL = fs.Arg(0)
	cfg.Headers = map[string]string(headers)
	cfg.BodyFile = bodyFile
	cfg.ExpectedStatus = []int(statuses)

	var thresholds compare.RunThresholds
	if p99UnderText != "" {
		thresholds.P99Under, err = timeutil.ParseDuration(p99UnderText)
		if err != nil {
			fmt.Fprintf(stderr, "invalid p99-under: %v\n", err)
			return 2
		}
	}
	if errorRateUnderText != "" {
		thresholds.ErrorRateUnder, err = timeutil.ParsePercentThreshold(errorRateUnderText)
		if err != nil {
			fmt.Fprintf(stderr, "invalid error-rate-under: %v\n", err)
			return 2
		}
	}
	runtimeTimeout := 5 * time.Second
	if runtimeTimeoutText != "" {
		runtimeTimeout, err = timeutil.ParseDuration(runtimeTimeoutText)
		if err != nil {
			fmt.Fprintf(stderr, "invalid runtime-timeout: %v\n", err)
			return 2
		}
	}
	if err := requirePositiveDuration("runtime-timeout", runtimeTimeout); err != nil {
		fmt.Fprintf(stderr, "invalid runtime-timeout: %v\n", err)
		return 2
	}

	var body []byte
	if bodyFile != "" {
		body, err = os.ReadFile(bodyFile)
		if err != nil {
			fmt.Fprintf(stderr, "read body file: %v\n", err)
			return 2
		}
		cfg.RequestBodySize = len(body)
	}

	var result probe.RunResult
	if runtimeURL != "" {
		before, after, err := collectRuntimeWindow(context.Background(), runtimeURL, runtimeTimeout, func() error {
			var runErr error
			result, runErr = probe.HTTPRunner{Config: cfg, Body: body}.Run(context.Background())
			return runErr
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		correlation := runtimesignal.Correlate(before, after)
		result.Runtime = &correlation
	} else {
		result, err = probe.HTTPRunner{Config: cfg, Body: body}.Run(context.Background())
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	output.WriteSummary(stdout, result)
	if outputPath != "" {
		if err := output.WriteJSONFile(outputPath, result); err != nil {
			fmt.Fprintf(stderr, "write json: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s\n", outputPath)
	}
	for _, export := range []struct {
		path   string
		format string
	}{
		{markdownOutput, "markdown"},
		{prometheusOutput, "prometheus"},
		{otelOutput, "otel"},
	} {
		if export.path == "" {
			continue
		}
		if err := writeFile(export.path, func(w io.Writer) error { return writeRunFormat(w, export.format, result) }); err != nil {
			fmt.Fprintf(stderr, "write %s: %v\n", export.format, err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s\n", export.path)
	}

	failures := compare.EvaluateRun(result, thresholds)
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintf(stderr, "threshold failed: %s\n", failure)
		}
		return 1
	}
	return 0
}
