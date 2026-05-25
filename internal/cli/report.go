package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/Justin-Arnold/p99/internal/output"
)

func runReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	var format, outputPath string
	var details, histogram, slowSamples, shape, runtime bool
	fs.StringVar(&format, "format", "text", "output format: text, json, markdown, prometheus, or otel")
	fs.StringVar(&outputPath, "output", "", "write output to path instead of stdout")
	fs.BoolVar(&details, "details", false, "include histogram, slow samples, shape detail, and runtime sections")
	fs.BoolVar(&histogram, "histogram", false, "include histogram buckets")
	fs.BoolVar(&slowSamples, "slow-samples", false, "include retained slow request samples")
	fs.BoolVar(&shape, "shape", false, "include detailed shape analysis")
	fs.BoolVar(&runtime, "runtime", false, "include runtime correlation if present")
	if err := parse(fs, args, map[string]bool{"format": true, "output": true}); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "report requires exactly one run JSON file")
		return 2
	}
	result, err := output.ReadJSON(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "read report: %v\n", err)
		return 1
	}
	opts := output.ReportOptions{
		Details:     details,
		Histogram:   histogram,
		SlowSamples: slowSamples,
		Shape:       shape,
		Runtime:     runtime,
	}
	if outputPath != "" {
		if err := writeFile(outputPath, func(w io.Writer) error { return writeReportFormat(w, format, result, opts) }); err != nil {
			fmt.Fprintf(stderr, "write report: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeReportFormat(stdout, format, result, opts); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}
