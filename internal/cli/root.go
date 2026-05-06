package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "http":
		return runHTTP(args[1:], stdout, stderr)
	case "watch":
		return runWatch(args[1:], stdout, stderr)
	case "profile":
		return runProfile(args[1:], stdout, stderr)
	case "runtime":
		return runRuntime(args[1:], stdout, stderr)
	case "spans":
		return runSpans(args[1:], stdout, stderr)
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "compare":
		return runCompare(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  p99 http [flags] URL")
	fmt.Fprintln(w, "  p99 watch [flags] URL")
	fmt.Fprintln(w, "  p99 profile [flags] PPROF_URL")
	fmt.Fprintln(w, "  p99 runtime [flags] BASE_URL")
	fmt.Fprintln(w, "  p99 spans [flags] traces.json")
	fmt.Fprintln(w, "  p99 report run.json")
	fmt.Fprintln(w, "  p99 compare [flags] before.json after.json")
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
