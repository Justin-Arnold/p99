package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/spans"
)

func runSpans(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("spans", flag.ContinueOnError)
	var jsonOut string
	var slowSamples int
	var format string
	fs.StringVar(&jsonOut, "json-out", "", "write span report JSON to path")
	fs.IntVar(&slowSamples, "slow-samples", 10, "number of slow traces to retain")
	fs.StringVar(&format, "format", "text", "terminal output format: text or json")
	if err := parse(fs, args, map[string]bool{"json-out": true, "slow-samples": true, "format": true}); err != nil {
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

	switch format {
	case "text":
		output.WriteSpanReport(stdout, report)
	case "json":
		if err := writeSpanJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "write span json: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unknown format %q\n", format)
		return 2
	}

	if jsonOut != "" {
		f, err := os.Create(jsonOut)
		if err != nil {
			fmt.Fprintf(stderr, "write span json: %v\n", err)
			return 1
		}
		if err := writeSpanJSON(f, report); err != nil {
			_ = f.Close()
			fmt.Fprintf(stderr, "write span json: %v\n", err)
			return 1
		}
		if err := f.Close(); err != nil {
			fmt.Fprintf(stderr, "write span json: %v\n", err)
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
