package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/justin/p99/internal/spans"
)

func runSpans(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("spans", flag.ContinueOnError)
	var jsonOut string
	var markdownOut, prometheusOut, otelOut string
	var slowSamples int
	var format string
	fs.StringVar(&jsonOut, "json-out", "", "write span report JSON to path")
	fs.StringVar(&markdownOut, "markdown-out", "", "write Markdown span report to path")
	fs.StringVar(&prometheusOut, "prometheus-out", "", "write Prometheus text metrics to path")
	fs.StringVar(&otelOut, "otel-out", "", "write OpenTelemetry metrics JSON to path")
	fs.IntVar(&slowSamples, "slow-samples", 10, "number of slow traces to retain")
	fs.StringVar(&format, "format", "text", "terminal output format: text, json, markdown, prometheus, or otel")
	if err := parse(fs, args, map[string]bool{"json-out": true, "markdown-out": true, "prometheus-out": true, "otel-out": true, "slow-samples": true, "format": true}); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "spans requires exactly one JSON input file")
		return 2
	}
	source := fs.Arg(0)
	var r io.Reader
	if source == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(source)
		if err != nil {
			fmt.Fprintf(stderr, "read spans: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	}

	parsed, warnings, err := spans.ParseJSON(r)
	if err != nil {
		fmt.Fprintf(stderr, "parse spans: %v\n", err)
		return 1
	}
	report := spans.Analyze(parsed, warnings, spans.AnalyzeOptions{Source: source, SlowSamples: slowSamples})

	if err := writeSpanFormat(stdout, format, report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	if jsonOut != "" {
		if err := writeFile(jsonOut, func(w io.Writer) error { return writeSpanFormat(w, "json", report) }); err != nil {
			fmt.Fprintf(stderr, "write span json: %v\n", err)
			return 1
		}
	}
	for _, export := range []struct {
		path   string
		format string
	}{
		{markdownOut, "markdown"},
		{prometheusOut, "prometheus"},
		{otelOut, "otel"},
	} {
		if export.path == "" {
			continue
		}
		if err := writeFile(export.path, func(w io.Writer) error { return writeSpanFormat(w, export.format, report) }); err != nil {
			fmt.Fprintf(stderr, "write span %s: %v\n", export.format, err)
			return 1
		}
	}
	return 0
}

func writeSpanJSON(w io.Writer, report spans.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
