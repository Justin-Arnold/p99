package main

import (
	"os"

	"github.com/Justin-Arnold/p99/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
