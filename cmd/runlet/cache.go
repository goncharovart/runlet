package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/goncharovart/runlet/internal/cachemgr"
)

// runCache dispatches the `runlet cache <subcommand>` family. It is
// kept in its own file so the main.go entry point reads as a flat
// switch and so the subcommand body stays close to its tests.
func runCache(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(argv) == 0 {
		fmt.Fprintln(stderr, "runlet cache: subcommand required (info | clear)")
		return 2
	}

	root, err := cachemgr.ResolveRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "runlet: %v\n", err)
		return 1
	}

	switch argv[0] {
	case "info":
		info, err := cachemgr.Inspect(root)
		if err != nil {
			fmt.Fprintf(stderr, "runlet: %v\n", err)
			return 1
		}
		if err := cachemgr.PrintInfo(stdout, info); err != nil {
			fmt.Fprintf(stderr, "runlet: %v\n", err)
			return 1
		}
		return 0

	case "clear":
		yes := false
		for _, a := range argv[1:] {
			if a == "--yes" || a == "-y" {
				yes = true
			}
		}
		info, err := cachemgr.Inspect(root)
		if err != nil {
			fmt.Fprintf(stderr, "runlet: %v\n", err)
			return 1
		}
		if !yes {
			fmt.Fprintf(stdout, "About to wipe %s (%d scripts). Confirm? [y/N] ", info.Root, info.Scripts)
			reader := bufio.NewReader(stdin)
			line, _ := reader.ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(line), "y") {
				fmt.Fprintln(stdout, "aborted")
				return 0
			}
		}
		if err := cachemgr.Clear(root); err != nil {
			fmt.Fprintf(stderr, "runlet: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "cache cleared")
		return 0

	default:
		fmt.Fprintf(stderr, "runlet cache: unknown subcommand %q\n", argv[0])
		return 2
	}
}
