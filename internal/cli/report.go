package cli

import (
	"fmt"
	"io"

	"github.com/justin/p99/internal/output"
)

func runReport(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "report requires exactly one run JSON file")
		return 2
	}
	result, err := output.ReadJSON(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "read report: %v\n", err)
		return 1
	}
	output.WriteSummary(stdout, result)
	return 0
}
