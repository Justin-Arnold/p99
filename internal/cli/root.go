package cli

import (
	"flag"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type commandSpec struct {
	Name    string
	Usage   string
	Summary string
	Flags   []flagSpec
	Run     func([]string, io.Writer, io.Writer) int
}

type flagSpec struct {
	Long        string
	Short       string
	TakesValue  bool
	Description string
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "-h", "--help":
		usage(stdout)
		return 0
	case "-v", "--version", "version":
		writeVersion(stdout)
		return 0
	case "help":
		return runHelp(args[1:], stdout, stderr)
	case "completion":
		return runCompletion(args[1:], stdout, stderr)
	}

	spec, ok := commandByName(args[0])
	if !ok {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
	if containsHelp(args[1:]) {
		commandUsage(stdout, spec)
		return 0
	}
	return spec.Run(args[1:], stdout, stderr)
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "help accepts at most one command")
		return 2
	}
	if args[0] == "completion" {
		completionUsage(stdout)
		return 0
	}
	if args[0] == "version" {
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  p99 version")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Print build version, commit, and build date.")
		return 0
	}
	spec, ok := commandByName(args[0])
	if !ok {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
	commandUsage(stdout, spec)
	return 0
}

func writeVersion(w io.Writer) {
	fmt.Fprintf(w, "p99 %s\n", Version)
	fmt.Fprintf(w, "commit: %s\n", Commit)
	fmt.Fprintf(w, "built: %s\n", Date)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "p99 profiles tail latency from HTTP probes, saved runs, runtime signals, profiles, and spans.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "  %s\n", spec.Usage)
	}
	fmt.Fprintln(w, "  p99 version")
	fmt.Fprintln(w, "  p99 completion SHELL")
	fmt.Fprintln(w, "  p99 help [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "  %-10s %s\n", spec.Name, spec.Summary)
	}
	fmt.Fprintln(w, "  version    Print build version information.")
	fmt.Fprintln(w, "  completion Generate shell completion scripts for bash, zsh, or fish.")
}

func commandUsage(w io.Writer, spec commandSpec) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", spec.Usage)
	fmt.Fprintln(w)
	fmt.Fprintln(w, spec.Summary)
	if len(spec.Flags) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	for _, f := range spec.Flags {
		fmt.Fprintf(w, "  %-24s %s\n", f.display(), f.Description)
	}
}

func completionUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  p99 completion bash")
	fmt.Fprintln(w, "  p99 completion zsh")
	fmt.Fprintln(w, "  p99 completion fish")
}

func containsHelp(args []string) bool {
	for i, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			if arg == "help" && i != 0 {
				continue
			}
			return true
		}
	}
	return false
}

func commandByName(name string) (commandSpec, bool) {
	for _, spec := range commandSpecs() {
		if spec.Name == name {
			return spec, true
		}
	}
	return commandSpec{}, false
}

func commandSpecs() []commandSpec {
	return []commandSpec{
		{
			Name:    "http",
			Usage:   "p99 http [flags] URL",
			Summary: "Probe an HTTP endpoint and produce latency/error summaries.",
			Run:     runHTTP,
			Flags: []flagSpec{
				{Long: "duration", TakesValue: true, Description: "Measured probe duration."},
				{Long: "rps", TakesValue: true, Description: "Target requests per second."},
				{Long: "concurrency", TakesValue: true, Description: "Maximum concurrent requests."},
				{Long: "warmup", TakesValue: true, Description: "Warmup duration before measurements are recorded."},
				{Long: "timeout", TakesValue: true, Description: "Per-request timeout."},
				{Long: "method", TakesValue: true, Description: "HTTP method."},
				{Short: "H", TakesValue: true, Description: "Request header, Name: value."},
				{Long: "header", TakesValue: true, Description: "Request header, Name: value."},
				{Long: "body-file", TakesValue: true, Description: "File to use as request body."},
				{Long: "status", TakesValue: true, Description: "Expected status code; may be repeated or comma-separated."},
				{Long: "output", TakesValue: true, Description: "Write run JSON to path."},
				{Long: "out", TakesValue: true, Description: "Alias for --output."},
				{Long: "slow-samples", TakesValue: true, Description: "Number of slow request samples to retain."},
				{Long: "p99-under", TakesValue: true, Description: "Fail if p99 is above duration."},
				{Long: "error-rate-under", TakesValue: true, Description: "Fail if error rate is above percent."},
				{Long: "runtime", TakesValue: true, Description: "Go runtime base URL for correlation."},
				{Long: "runtime-timeout", TakesValue: true, Description: "Runtime endpoint timeout."},
				{Long: "markdown-output", TakesValue: true, Description: "Write Markdown report to path."},
				{Long: "prometheus-output", TakesValue: true, Description: "Write Prometheus text metrics to path."},
				{Long: "otel-output", TakesValue: true, Description: "Write OpenTelemetry metrics JSON to path."},
			},
		},
		{
			Name:    "watch",
			Usage:   "p99 watch [flags] URL",
			Summary: "Continuously run short probe windows and redraw a live summary.",
			Run:     runWatch,
			Flags: []flagSpec{
				{Long: "window", TakesValue: true, Description: "Duration for each dashboard refresh."},
				{Long: "duration", TakesValue: true, Description: "Alias for --window."},
				{Long: "rps", TakesValue: true, Description: "Target requests per second."},
				{Long: "concurrency", TakesValue: true, Description: "Maximum concurrent requests."},
				{Long: "timeout", TakesValue: true, Description: "Per-request timeout."},
				{Long: "method", TakesValue: true, Description: "HTTP method."},
				{Short: "H", TakesValue: true, Description: "Request header, Name: value."},
				{Long: "header", TakesValue: true, Description: "Request header, Name: value."},
				{Long: "body-file", TakesValue: true, Description: "File to use as request body."},
				{Long: "status", TakesValue: true, Description: "Expected status code; may be repeated or comma-separated."},
				{Long: "slow-samples", TakesValue: true, Description: "Number of slow request samples to retain."},
				{Long: "iterations", TakesValue: true, Description: "Number of windows to run; 0 runs until interrupted."},
				{Long: "clear", Description: "Clear the terminal before each refresh."},
				{Long: "runtime", TakesValue: true, Description: "Go runtime base URL for correlation."},
				{Long: "runtime-timeout", TakesValue: true, Description: "Runtime endpoint timeout."},
			},
		},
		{
			Name:    "profile",
			Usage:   "p99 profile [flags] PPROF_URL",
			Summary: "Capture a Go pprof profile and optionally correlate it with probe/runtime data.",
			Run:     runProfile,
			Flags: []flagSpec{
				{Long: "seconds", TakesValue: true, Description: "Seconds to capture from the pprof endpoint."},
				{Long: "timeout", TakesValue: true, Description: "Profile request timeout."},
				{Long: "output", TakesValue: true, Description: "Write pprof data to path."},
				{Long: "out", TakesValue: true, Description: "Alias for --output."},
				{Long: "top", Description: "Run go tool pprof -top after capture."},
				{Long: "top-count", TakesValue: true, Description: "Number of pprof top rows."},
				{Long: "probe", TakesValue: true, Description: "HTTP URL to probe while the profile is captured."},
				{Long: "probe-duration", TakesValue: true, Description: "HTTP probe duration; defaults to profile seconds."},
				{Long: "probe-rps", TakesValue: true, Description: "HTTP probe requests per second."},
				{Long: "probe-concurrency", TakesValue: true, Description: "HTTP probe concurrency."},
				{Long: "runtime", TakesValue: true, Description: "Go runtime base URL for correlation."},
				{Long: "runtime-timeout", TakesValue: true, Description: "Runtime endpoint timeout."},
			},
		},
		{
			Name:    "runtime",
			Usage:   "p99 runtime [flags] BASE_URL",
			Summary: "Collect Go runtime signal movement over a time window.",
			Run:     runRuntime,
			Flags: []flagSpec{
				{Long: "duration", TakesValue: true, Description: "Runtime correlation window."},
				{Long: "timeout", TakesValue: true, Description: "Runtime endpoint timeout."},
			},
		},
		{
			Name:    "spans",
			Usage:   "p99 spans [flags] traces.json",
			Summary: "Analyze OpenTelemetry-style span JSON for route and dependency timing.",
			Run:     runSpans,
			Flags: []flagSpec{
				{Long: "json-out", TakesValue: true, Description: "Write span report JSON to path."},
				{Long: "markdown-out", TakesValue: true, Description: "Write Markdown span report to path."},
				{Long: "prometheus-out", TakesValue: true, Description: "Write Prometheus text metrics to path."},
				{Long: "otel-out", TakesValue: true, Description: "Write OpenTelemetry metrics JSON to path."},
				{Long: "slow-samples", TakesValue: true, Description: "Number of slow traces to retain."},
				{Long: "format", TakesValue: true, Description: "Terminal output format: text, json, markdown, prometheus, or otel."},
			},
		},
		{
			Name:    "report",
			Usage:   "p99 report [flags] run.json",
			Summary: "Read a saved run JSON file and print or export a report.",
			Run:     runReport,
			Flags: []flagSpec{
				{Long: "format", TakesValue: true, Description: "Output format: text, json, markdown, prometheus, or otel."},
				{Long: "output", TakesValue: true, Description: "Write output to path instead of stdout."},
				{Long: "details", Description: "Include histogram, slow samples, shape detail, and runtime sections."},
				{Long: "histogram", Description: "Include histogram buckets."},
				{Long: "slow-samples", Description: "Include retained slow request samples."},
				{Long: "shape", Description: "Include detailed shape analysis."},
				{Long: "runtime", Description: "Include runtime correlation if present."},
			},
		},
		{
			Name:    "compare",
			Usage:   "p99 compare [flags] before.json after.json",
			Summary: "Compare two saved run JSON files and enforce budgets.",
			Run:     runCompare,
			Flags: []flagSpec{
				{Long: "max-p50-regression", TakesValue: true, Description: "Fail if p50 regression exceeds percent."},
				{Long: "max-p95-regression", TakesValue: true, Description: "Fail if p95 regression exceeds percent."},
				{Long: "max-p99-regression", TakesValue: true, Description: "Fail if p99 regression exceeds percent."},
				{Long: "max-p999-regression", TakesValue: true, Description: "Fail if p999 regression exceeds percent."},
				{Long: "max-max-regression", TakesValue: true, Description: "Fail if max latency regression exceeds percent."},
				{Long: "max-error-rate-regression", TakesValue: true, Description: "Fail if error rate regression exceeds percent."},
				{Long: "error-rate-under", TakesValue: true, Description: "Fail if after error rate exceeds percent."},
				{Long: "max-error-rate", TakesValue: true, Description: "Alias for --error-rate-under."},
				{Long: "min-request-count", TakesValue: true, Description: "Fail if after request count is below this value."},
				{Long: "max-request-drop", TakesValue: true, Description: "Fail if request count drop exceeds percent."},
				{Long: "p50-under", TakesValue: true, Description: "Fail if after p50 exceeds a duration."},
				{Long: "p95-under", TakesValue: true, Description: "Fail if after p95 exceeds a duration."},
				{Long: "p99-under", TakesValue: true, Description: "Fail if after p99 exceeds a duration."},
				{Long: "p999-under", TakesValue: true, Description: "Fail if after p999 exceeds a duration."},
				{Long: "max-under", TakesValue: true, Description: "Fail if after max latency exceeds a duration."},
			},
		},
	}
}

func (f flagSpec) display() string {
	parts := []string{}
	if f.Short != "" {
		short := "-" + f.Short
		if f.TakesValue {
			short += " value"
		}
		parts = append(parts, short)
	}
	if f.Long != "" {
		long := "--" + f.Long
		if f.TakesValue {
			long += " value"
		}
		parts = append(parts, long)
	}
	return strings.Join(parts, ", ")
}

func parse(fs *flag.FlagSet, args []string, valueFlags map[string]bool) error {
	fs.SetOutput(io.Discard)
	return fs.Parse(reorderArgs(args, valueFlags))
}

func reorderArgs(args []string, valueFlags map[string]bool) []string {
	flags := []string{}
	positionals := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			name := strings.TrimLeft(arg, "-")
			if idx := strings.IndexByte(name, '='); idx >= 0 {
				name = name[:idx]
			}
			if !strings.Contains(arg, "=") && valueFlags[name] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}

func requirePositiveDuration(name string, d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}

func requirePositiveFloat(name string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%s must be finite", name)
	}
	if v <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}

func requirePositiveInt(name string, v int) error {
	if v <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}

func requireNonNegativeInt(name string, v int) error {
	if v < 0 {
		return fmt.Errorf("%s must be non-negative", name)
	}
	return nil
}
