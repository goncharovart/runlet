package main

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
)

// versionString assembles the one-line --version output. It is
// extracted so the same logic feeds both the live --version handler
// and the unit test (the test injects a fake debug.ReadBuildInfo
// result so we don't depend on whether the test binary itself was
// built from a tagged release).
type buildFacts struct {
	main      string // main module version, e.g. "v0.1.0" or "(devel)"
	revision  string // VCS commit hash; empty when missing
	dirty     bool   // VCS state was modified at build time
	buildTime string // VCS build time when available
	goVersion string // Go toolchain version, e.g. "go1.25.0"
}

func versionString(b buildFacts) string {
	out := "runlet " + b.main
	if b.revision != "" {
		short := b.revision
		if len(short) > 12 {
			short = short[:12]
		}
		out += " (" + short
		if b.dirty {
			out += "-dirty"
		}
		out += ")"
	}
	if b.buildTime != "" {
		out += " built " + b.buildTime
	}
	out += " · " + b.goVersion
	return out
}

// printVersion gathers build facts from runtime/debug.ReadBuildInfo
// (the modern Go-1.18+ way; no -ldflags -X required) and writes the
// summary to w. Returns nil on success and an error only when w.Write
// fails.
//
// When the binary was not built from a tracked module (the common
// `go run cmd/runlet/main.go --version` case during development),
// ReadBuildInfo returns "(devel)" / "" defaults — the helper still
// produces a useful one-liner.
func printVersion(w io.Writer) error {
	facts := buildFacts{
		main:      "(devel)",
		goVersion: runtime.Version(),
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" {
			facts.main = info.Main.Version
		}
		if info.GoVersion != "" {
			facts.goVersion = info.GoVersion
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				facts.revision = s.Value
			case "vcs.time":
				facts.buildTime = s.Value
			case "vcs.modified":
				facts.dirty = s.Value == "true"
			}
		}
	}
	_, err := fmt.Fprintln(w, versionString(facts))
	return err
}
