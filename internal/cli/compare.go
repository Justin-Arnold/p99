package cli

import (
	"flag"
	"fmt"
	"io"

	p99compare "github.com/justin/p99/internal/compare"
	"github.com/justin/p99/internal/output"
	"github.com/justin/p99/internal/timeutil"
)

func runCompare(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	var maxP99RegressionText string
	fs.StringVar(&maxP99RegressionText, "max-p99-regression", "", "fail if p99 regression exceeds percent")
	if err := parse(fs, args, map[string]bool{"max-p99-regression": true}); err != nil {
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

	var thresholds p99compare.CompareThresholds
	if maxP99RegressionText != "" {
		thresholds.MaxP99Regression, err = timeutil.ParsePercentThreshold(maxP99RegressionText)
		if err != nil {
			fmt.Fprintf(stderr, "invalid max-p99-regression: %v\n", err)
			return 2
		}
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
