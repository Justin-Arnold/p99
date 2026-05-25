package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Justin-Arnold/p99/internal/output"
	"github.com/Justin-Arnold/p99/internal/probe"
	p99profile "github.com/Justin-Arnold/p99/internal/profile"
	"github.com/Justin-Arnold/p99/internal/runtimesignal"
	"github.com/Justin-Arnold/p99/internal/timeutil"
)

func runProfile(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("profile", flag.ContinueOnError)
	var timeoutText, outputPath, probeURL string
	var probeDurationText string
	var runtimeURL, runtimeTimeoutText string
	var probeRPS float64
	var probeConcurrency int
	cfg := p99profile.Config{Seconds: 30, Timeout: 45 * time.Second, Top: true, TopCount: 10, UserAgent: "p99"}
	fs.IntVar(&cfg.Seconds, "seconds", 30, "seconds to capture from the pprof endpoint")
	fs.StringVar(&timeoutText, "timeout", "45s", "profile request timeout")
	fs.StringVar(&outputPath, "output", "", "write pprof data to path")
	fs.StringVar(&outputPath, "out", "", "write pprof data to path")
	fs.BoolVar(&cfg.Top, "top", true, "run go tool pprof -top after capture")
	fs.IntVar(&cfg.TopCount, "top-count", 10, "number of pprof top rows")
	fs.StringVar(&probeURL, "probe", "", "HTTP URL to probe while the profile is captured")
	fs.StringVar(&probeDurationText, "probe-duration", "", "HTTP probe duration; defaults to profile seconds")
	fs.Float64Var(&probeRPS, "probe-rps", 1, "HTTP probe requests per second")
	fs.IntVar(&probeConcurrency, "probe-concurrency", 1, "HTTP probe concurrency")
	fs.StringVar(&runtimeURL, "runtime", "", "Go runtime base URL for correlation, e.g. http://localhost:8080")
	fs.StringVar(&runtimeTimeoutText, "runtime-timeout", "5s", "runtime endpoint timeout")

	valueFlags := map[string]bool{
		"seconds": true, "timeout": true, "output": true, "out": true, "top-count": true,
		"probe": true, "probe-duration": true, "probe-rps": true, "probe-concurrency": true,
		"runtime": true, "runtime-timeout": true,
	}
	if err := parse(fs, args, valueFlags); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "profile requires exactly one pprof URL")
		return 2
	}
	var err error
	cfg.Timeout, err = timeutil.ParseDuration(timeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid timeout: %v\n", err)
		return 2
	}
	cfg.URL = fs.Arg(0)
	cfg.Output = outputPath
	runtimeTimeout, err := timeutil.ParseDuration(runtimeTimeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid runtime-timeout: %v\n", err)
		return 2
	}
	if err := requirePositiveInt("seconds", cfg.Seconds); err != nil {
		fmt.Fprintf(stderr, "invalid seconds: %v\n", err)
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
	if err := requirePositiveInt("top-count", cfg.TopCount); err != nil {
		fmt.Fprintf(stderr, "invalid top-count: %v\n", err)
		return 2
	}
	if err := requirePositiveFloat("probe-rps", probeRPS); err != nil {
		fmt.Fprintf(stderr, "invalid probe-rps: %v\n", err)
		return 2
	}
	if err := requirePositiveInt("probe-concurrency", probeConcurrency); err != nil {
		fmt.Fprintf(stderr, "invalid probe-concurrency: %v\n", err)
		return 2
	}

	var probeResult probe.RunResult
	var haveProbe bool
	var runtimeCorrelation *runtimesignal.Correlation
	if probeURL != "" {
		duration := time.Duration(cfg.Seconds) * time.Second
		if probeDurationText != "" {
			duration, err = timeutil.ParseDuration(probeDurationText)
			if err != nil {
				fmt.Fprintf(stderr, "invalid probe-duration: %v\n", err)
				return 2
			}
		}
		probeCfg := probe.HTTPConfig{
			URL:         probeURL,
			Method:      http.MethodGet,
			Duration:    duration,
			RPS:         probeRPS,
			Concurrency: probeConcurrency,
			Timeout:     cfg.Timeout,
			SlowSamples: 5,
		}
		type probeOutcome struct {
			result probe.RunResult
			err    error
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		probeDone := make(chan probeOutcome, 1)
		go func() {
			result, err := probe.HTTPRunner{Config: probeCfg}.Run(ctx)
			probeDone <- probeOutcome{result: result, err: err}
		}()

		var result p99profile.Result
		var code int
		if runtimeURL != "" {
			before, after, err := collectRuntimeWindow(ctx, runtimeURL, runtimeTimeout, func() error {
				var captureCode int
				result, captureCode = captureProfile(ctx, cfg, stdout, stderr)
				if captureCode != 0 {
					return fmt.Errorf("profile capture failed")
				}
				return nil
			})
			if err != nil {
				cancel()
				<-probeDone
				fmt.Fprintf(stderr, "runtime: %v\n", err)
				return 2
			}
			correlation := runtimesignal.Correlate(before, after)
			runtimeCorrelation = &correlation
		} else {
			result, code = captureProfile(ctx, cfg, stdout, stderr)
		}
		if code != 0 {
			cancel()
			<-probeDone
			return code
		}
		outcome := <-probeDone
		if outcome.err != nil {
			fmt.Fprintf(stderr, "probe failed: %v\n", outcome.err)
			return 2
		}
		probeResult = outcome.result
		haveProbe = true
		writeProfileSummary(stdout, result)
		if runtimeCorrelation != nil {
			fmt.Fprintln(stdout)
			output.WriteRuntime(stdout, *runtimeCorrelation)
		}
		if haveProbe {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Correlated HTTP probe")
			output.WriteSummary(stdout, probeResult)
		}
		return 0
	}

	var result p99profile.Result
	var code int
	if runtimeURL != "" {
		before, after, err := collectRuntimeWindow(context.Background(), runtimeURL, runtimeTimeout, func() error {
			var captureCode int
			result, captureCode = captureProfile(context.Background(), cfg, stdout, stderr)
			if captureCode != 0 {
				return fmt.Errorf("profile capture failed")
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "runtime: %v\n", err)
			return 2
		}
		correlation := runtimesignal.Correlate(before, after)
		runtimeCorrelation = &correlation
	} else {
		result, code = captureProfile(context.Background(), cfg, stdout, stderr)
	}
	if code != 0 {
		return code
	}
	writeProfileSummary(stdout, result)
	if runtimeCorrelation != nil {
		fmt.Fprintln(stdout)
		output.WriteRuntime(stdout, *runtimeCorrelation)
	}
	return 0
}

func captureProfile(ctx context.Context, cfg p99profile.Config, stdout, stderr io.Writer) (p99profile.Result, int) {
	result, err := p99profile.Fetcher{Config: cfg}.Fetch(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "capture profile: %v\n", err)
		return p99profile.Result{}, 1
	}
	return result, 0
}

func writeProfileSummary(w io.Writer, result p99profile.Result) {
	fmt.Fprintf(w, "Profile: %s\n", result.URL)
	fmt.Fprintf(w, "Saved: %s (%d bytes)\n", result.OutputPath, result.Bytes)
	fmt.Fprintf(w, "Captured: %s to %s\n", result.StartedAt.Format(time.RFC3339), result.EndedAt.Format(time.RFC3339))
	if result.Top != "" {
		fmt.Fprintln(w)
		fmt.Fprint(w, result.Top)
	}
	if result.TopError != "" {
		fmt.Fprintf(w, "pprof top unavailable: %s\n", result.TopError)
	}
}
