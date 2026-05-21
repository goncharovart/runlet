// Command runlet runs a single-file Go script that declares its
// dependencies through magic comments.
//
//	runlet hello.go
//	runlet hello.go arg1 arg2 ...
//	cat hello.go | runlet -
//
// The "-" form reads the script from stdin. Anything after the script
// path is forwarded as arguments to the script itself.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/goncharovart/runlet/internal/parser"
	"github.com/goncharovart/runlet/internal/runner"
)

const usage = `runlet — run a Go script with inline dependencies

usage:
    runlet [flags] <script.go> [args...]
    runlet -                   # read script from stdin
    runlet cache info          # show cache size + script count
    runlet cache clear [--yes] # wipe the cache directory

flags:
    -h, --help          show this help
    -v, --version       show build info

A script declares dependencies with magic comments at the top:

    // runlet:dep github.com/spf13/cobra v1.9.0
    package main

    func main() { ... }
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	switch argv[0] {
	case "-h", "--help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	case "-v", "--version":
		if err := printVersion(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "runlet: %v\n", err)
			return 1
		}
		return 0
	case "cache":
		return runCache(argv[1:], os.Stdin, os.Stdout, os.Stderr)
	}

	scriptPath := argv[0]
	scriptArgs := argv[1:]

	var (
		source io.Reader
		path   = scriptPath
	)
	if scriptPath == "-" {
		source = os.Stdin
		path = "stdin"
	} else {
		f, err := os.Open(scriptPath) //nolint:gosec
		if err != nil {
			fmt.Fprintf(os.Stderr, "runlet: open %s: %v\n", scriptPath, err)
			return 1
		}
		defer f.Close()
		source = f
	}

	script, err := parser.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "runlet: parse %s: %v\n", path, err)
		return 1
	}

	exit, err := runner.Run(runner.Config{
		Script:     script,
		ScriptPath: scriptPath,
		Args:       scriptArgs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "runlet: %v\n", err)
		return 1
	}
	return exit
}
