package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Justin-Arnold/p99/internal/output"
	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/runtimesignal"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

func runWatch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	var windowText, timeoutText, bodyFile string
	var runtimeURL, runtimeTimeoutText string
	var iterations int
	var clear bool
	headers := headerFlags{}
	statuses := statusFlags{}
	cfg := probe.HTTPConfig{Method: http.MethodGet, RPS: 1, Concurrency: 1, Timeout: 10 * time.Second, SlowSamples: 5}
	fs.StringVar(&windowText, "window", "5s", "duration for each dashboard refresh")
	fs.StringVar(&windowText, "duration", "5s", "duration for each dashboard refresh")
	fs.Float64Var(&cfg.RPS, "rps", 1, "target requests per second")
	fs.IntVar(&cfg.Concurrency, "concurrency", 1, "maximum concurrent requests")
	fs.StringVar(&timeoutText, "timeout", "10s", "request timeout")
	fs.StringVar(&cfg.Method, "method", http.MethodGet, "HTTP method")
	fs.Var(headers, "H", "request header, Name: value")
	fs.Var(headers, "header", "request header, Name: value")
	fs.StringVar(&bodyFile, "body-file", "", "file to use as request body")
	fs.Var(&statuses, "status", "expected status code; may be repeated or comma-separated")
	fs.IntVar(&cfg.SlowSamples, "slow-samples", 5, "number of slow request samples to retain")
	fs.IntVar(&iterations, "iterations", 0, "number of windows to run; 0 runs until interrupted")
	fs.BoolVar(&clear, "clear", true, "clear the terminal before each refresh")
	fs.StringVar(&runtimeURL, "runtime", "", "Go runtime base URL for correlation, e.g. http://localhost:8080")
	fs.StringVar(&runtimeTimeoutText, "runtime-timeout", "5s", "runtime endpoint timeout")

	valueFlags := map[string]bool{
		"window": true, "duration": true, "rps": true, "concurrency": true, "timeout": true,
		"method": true, "H": true, "header": true, "body-file": true, "status": true,
		"slow-samples": true, "iterations": true, "runtime": true, "runtime-timeout": true,
	}
	if err := parse(fs, args, valueFlags); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "watch requires exactly one URL")
		return 2
	}
	window, err := timeutil.ParseDuration(windowText)
	if err != nil || window <= 0 {
		fmt.Fprintf(stderr, "invalid window: %v\n", err)
		return 2
	}
	cfg.Timeout, err = timeutil.ParseDuration(timeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid timeout: %v\n", err)
		return 2
	}
	runtimeTimeout, err := timeutil.ParseDuration(runtimeTimeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid runtime-timeout: %v\n", err)
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
	if err := requirePositiveDuration("runtime-timeout", runtimeTimeout); err != nil {
		fmt.Fprintf(stderr, "invalid runtime-timeout: %v\n", err)
		return 2
	}
	if err := requireNonNegativeInt("slow-samples", cfg.SlowSamples); err != nil {
		fmt.Fprintf(stderr, "invalid slow-samples: %v\n", err)
		return 2
	}
	if err := requireNonNegativeInt("iterations", iterations); err != nil {
		fmt.Fprintf(stderr, "invalid iterations: %v\n", err)
		return 2
	}
	cfg.Duration = window
	cfg.URL = fs.Arg(0)
	cfg.Headers = map[string]string(headers)
	cfg.BodyFile = bodyFile
	cfg.ExpectedStatus = []int(statuses)

	var body []byte
	if bodyFile != "" {
		body, err = os.ReadFile(bodyFile)
		if err != nil {
			fmt.Fprintf(stderr, "read body file: %v\n", err)
			return 2
		}
		cfg.RequestBodySize = len(body)
	}

	for i := 0; iterations == 0 || i < iterations; i++ {
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
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
		}
		if clear {
			fmt.Fprint(stdout, "\033[H\033[2J")
		}
		fmt.Fprintf(stdout, "p99 watch  target=%s  window=%s  refresh=%d\n", cfg.URL, timeutil.FormatDuration(window), i+1)
		fmt.Fprintf(stdout, "Updated: %s\n\n", result.EndedAt.Format(time.RFC3339))
		output.WriteSummary(stdout, result)
		if iterations == 0 {
			fmt.Fprintln(stdout)
		}
	}
	return 0
}
