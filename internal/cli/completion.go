package cli

import (
	"fmt"
	"io"
	"strings"
)

func runCompletion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		completionUsage(stdout)
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
			return 0
		}
		return 2
	}
	switch args[0] {
	case "bash":
		writeBashCompletion(stdout)
	case "zsh":
		writeZshCompletion(stdout)
	case "fish":
		writeFishCompletion(stdout)
	default:
		fmt.Fprintf(stderr, "unsupported shell %q; expected bash, zsh, or fish\n", args[0])
		return 2
	}
	return 0
}

func writeBashCompletion(w io.Writer) {
	// The generated scripts are intentionally static. Package managers and Nix
	// modules can install them without the CLI mutating a user's shell files.
	fmt.Fprintln(w, `_p99_completion() {`)
	fmt.Fprintln(w, `  local cur cmd flags`)
	fmt.Fprintln(w, `  COMPREPLY=()`)
	fmt.Fprintln(w, `  cur="${COMP_WORDS[COMP_CWORD]}"`)
	fmt.Fprintln(w, `  if [[ ${COMP_CWORD} -eq 1 ]]; then`)
	fmt.Fprintf(w, "    COMPREPLY=( $(compgen -W %s -- \"$cur\") )\n", shellQuote(commandNames()))
	fmt.Fprintln(w, `    return 0`)
	fmt.Fprintln(w, `  fi`)
	fmt.Fprintln(w, `  cmd="${COMP_WORDS[1]}"`)
	fmt.Fprintln(w, `  case "$cmd" in`)
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "    %s) flags=%s ;;\n", spec.Name, shellQuote(flagWords(spec)))
	}
	fmt.Fprintln(w, `    completion) flags="bash zsh fish --help -h" ;;`)
	fmt.Fprintln(w, `    help) flags="`+commandNames()+`" ;;`)
	fmt.Fprintln(w, `    *) flags="" ;;`)
	fmt.Fprintln(w, `  esac`)
	fmt.Fprintln(w, `  COMPREPLY=( $(compgen -W "$flags" -- "$cur") )`)
	fmt.Fprintln(w, `}`)
	fmt.Fprintln(w, `complete -F _p99_completion p99`)
}

func writeZshCompletion(w io.Writer) {
	fmt.Fprintln(w, "#compdef p99")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "_p99() {")
	fmt.Fprintln(w, "  local -a commands")
	fmt.Fprintln(w, "  commands=(")
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "    %s\n", zshDescribe(spec.Name, spec.Summary))
	}
	fmt.Fprintf(w, "    %s\n", zshDescribe("version", "Print build version information."))
	fmt.Fprintf(w, "    %s\n", zshDescribe("completion", "Generate shell completion scripts."))
	fmt.Fprintf(w, "    %s\n", zshDescribe("help", "Show help for p99 or a command."))
	fmt.Fprintln(w, "  )")
	fmt.Fprintln(w, "  if (( CURRENT == 2 )); then")
	fmt.Fprintln(w, "    _describe -t commands 'p99 command' commands")
	fmt.Fprintln(w, "    return")
	fmt.Fprintln(w, "  fi")
	fmt.Fprintln(w, "  case $words[2] in")
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "    %s)\n", spec.Name)
		fmt.Fprintln(w, "      _arguments -s \\")
		for i, f := range append(spec.Flags, flagSpec{Long: "help", Short: "h", Description: "Show command help."}) {
			suffix := " \\"
			if i == len(spec.Flags) {
				suffix = ""
			}
			fmt.Fprintf(w, "        %s%s\n", shellQuote(zshArgument(f)), suffix)
		}
		fmt.Fprintln(w, "      ;;")
	}
	fmt.Fprintln(w, "    completion)")
	fmt.Fprintln(w, "      _arguments '1:shell:(bash zsh fish)'")
	fmt.Fprintln(w, "      ;;")
	fmt.Fprintln(w, "    help)")
	fmt.Fprintf(w, "      _arguments '1:command:(%s)'\n", commandNames())
	fmt.Fprintln(w, "      ;;")
	fmt.Fprintln(w, "  esac")
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "_p99 \"$@\"")
}

func writeFishCompletion(w io.Writer) {
	fmt.Fprintln(w, "complete -c p99 -f")
	for _, spec := range commandSpecs() {
		fmt.Fprintf(w, "complete -c p99 -n '__fish_use_subcommand' -a %s -d %s\n", shellQuote(spec.Name), shellQuote(trimPeriod(spec.Summary)))
		for _, f := range spec.Flags {
			fmt.Fprintf(w, "complete -c p99 -n '__fish_seen_subcommand_from %s' %s%s -d %s\n", spec.Name, fishFlag(f), fishRequiresValue(f), shellQuote(trimPeriod(f.Description)))
		}
		fmt.Fprintf(w, "complete -c p99 -n '__fish_seen_subcommand_from %s' -l help -s h -d 'Show command help'\n", spec.Name)
	}
	fmt.Fprintln(w, "complete -c p99 -n '__fish_use_subcommand' -a version -d 'Print build version information'")
	fmt.Fprintln(w, "complete -c p99 -n '__fish_use_subcommand' -a completion -d 'Generate shell completion scripts'")
	fmt.Fprintln(w, "complete -c p99 -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'")
	fmt.Fprintln(w, "complete -c p99 -n '__fish_use_subcommand' -a help -d 'Show help for p99 or a command'")
	fmt.Fprintf(w, "complete -c p99 -n '__fish_seen_subcommand_from help' -a %s\n", shellQuote(commandNames()))
}

func commandNames() string {
	names := []string{}
	for _, spec := range commandSpecs() {
		names = append(names, spec.Name)
	}
	names = append(names, "version", "completion", "help")
	return strings.Join(names, " ")
}

func flagWords(spec commandSpec) string {
	words := []string{"--help", "-h"}
	for _, f := range spec.Flags {
		if f.Long != "" {
			words = append(words, "--"+f.Long)
		}
		if f.Short != "" {
			words = append(words, "-"+f.Short)
		}
	}
	return strings.Join(words, " ")
}

func zshDescribe(name, description string) string {
	return shellQuote(name + ":" + trimPeriod(description))
}

func zshArgument(f flagSpec) string {
	if f.Long != "" {
		if f.TakesValue {
			return "--" + f.Long + "[" + trimPeriod(f.Description) + "]:value:"
		}
		return "--" + f.Long + "[" + trimPeriod(f.Description) + "]"
	}
	if f.Short != "" {
		if f.TakesValue {
			return "-" + f.Short + "[" + trimPeriod(f.Description) + "]:value:"
		}
		return "-" + f.Short + "[" + trimPeriod(f.Description) + "]"
	}
	return ""
}

func fishFlag(f flagSpec) string {
	parts := []string{}
	if f.Long != "" {
		parts = append(parts, "-l "+f.Long)
	}
	if f.Short != "" {
		parts = append(parts, "-s "+f.Short)
	}
	return strings.Join(parts, " ")
}

func fishRequiresValue(f flagSpec) string {
	if f.TakesValue {
		return " -r"
	}
	return ""
}

func trimPeriod(s string) string {
	return strings.TrimSuffix(s, ".")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
