package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/runtimesignal"
	"github.com/justin/p99/internal/timeutil"
)

func runRuntime(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("runtime", flag.ContinueOnError)
	var durationText, timeoutText string
	fs.StringVar(&durationText, "duration", "1s", "runtime correlation window")
	fs.StringVar(&timeoutText, "timeout", "5s", "runtime endpoint timeout")
	if err := parse(fs, args, map[string]bool{"duration": true, "timeout": true}); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "runtime requires exactly one base URL")
		return 2
	}
	duration, err := timeutil.ParseDuration(durationText)
	if err != nil || duration <= 0 {
		fmt.Fprintf(stderr, "invalid duration: %v\n", err)
		return 2
	}
	timeout, err := timeutil.ParseDuration(timeoutText)
	if err != nil {
		fmt.Fprintf(stderr, "invalid timeout: %v\n", err)
		return 2
	}

	before, after, err := collectRuntimeWindow(context.Background(), fs.Arg(0), timeout, func() error {
		time.Sleep(duration)
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "runtime: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Runtime: %s  window=%s\n", fs.Arg(0), timeutil.FormatDuration(duration))
	output.WriteRuntime(stdout, runtimesignal.Correlate(before, after))
	return 0
}

func collectRuntimeWindow(ctx context.Context, baseURL string, timeout time.Duration, fn func() error) (runtimesignal.Snapshot, runtimesignal.Snapshot, error) {
	collector := runtimesignal.Collector{Config: runtimesignal.Config{BaseURL: baseURL, Timeout: timeout}}
	before, err := collector.Collect(ctx)
	if err != nil {
		return runtimesignal.Snapshot{}, runtimesignal.Snapshot{}, err
	}
	if err := fn(); err != nil {
		return before, runtimesignal.Snapshot{}, err
	}
	after, err := collector.Collect(ctx)
	if err != nil {
		return before, runtimesignal.Snapshot{}, err
	}
	return before, after, nil
}
