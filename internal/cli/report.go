package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/justin/p99/internal/output"
)

func runReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	var format, outputPath string
	fs.StringVar(&format, "format", "text", "output format: text, json, markdown, prometheus, or otel")
	fs.StringVar(&outputPath, "output", "", "write output to path instead of stdout")
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
	if outputPath != "" {
		if err := writeFile(outputPath, func(w io.Writer) error { return writeRunFormat(w, format, result) }); err != nil {
			fmt.Fprintf(stderr, "write report: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeRunFormat(stdout, format, result); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}
