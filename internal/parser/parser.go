// Package parser extracts runlet metadata from a Go script.
//
// The metadata format is intentionally tiny: line comments of the form
//
//	// runlet:dep <module path> <version>
//
// declare third-party dependencies. Anything else is left for the Go
// toolchain to handle. A separator-less shebang trick on the first
// line is also recognised and stripped so the file remains parseable
// by `go run` and `go fmt`.
package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Dep is one declared dependency from a magic comment.
type Dep struct {
	Path    string // e.g. "github.com/spf13/cobra"
	Version string // e.g. "v1.9.0" or "latest"
}

// Script is the parsed view of a runlet-flavoured Go file.
type Script struct {
	// Source is the file content WITH the shebang line stripped (if
	// any). It is what runlet hands to `go run`.
	Source []byte

	// Deps is the ordered list of declared dependencies. Order
	// matches the appearance of the magic comments in the file so
	// the synthesised go.mod is deterministic for the same input.
	Deps []Dep

	// HadShebang is true when the first source line started with a
	// `#!` or `//usr/bin/env` shebang and was removed. Callers may
	// surface this in errors or telemetry.
	HadShebang bool
}

// magicPrefix is the comment marker that declares a dependency. The
// trailing-space variant is matched separately to surface "// runlet:dep"
// with no fields as an explicit error instead of silently ignoring it.
const magicPrefix = "// runlet:dep"

// Parse reads the entire script from r and returns the Script.
// Returns an error only when the input cannot be read or a magic
// comment is malformed — a script with no dependencies is valid.
func Parse(r io.Reader) (*Script, error) {
	br := bufio.NewReader(r)

	first, err := br.Peek(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("runlet/parser: peek: %w", err)
	}
	s := &Script{}

	// Strip a shebang-style first line. Two forms accepted:
	//   #!/usr/bin/env runlet
	//   //usr/bin/env runlet "$0" "$@"; exit
	// The second form is the Go-friendly trick: it parses as a Go
	// line comment, so `go fmt` and `go run` accept the file
	// unchanged. We strip it ourselves so a strict tool downstream
	// does not get confused.
	if len(first) >= 2 && (first[0] == '#' || (first[0] == '/' && first[1] == '/' && looksLikeShebangSecondForm(br))) {
		if _, _, err := br.ReadLine(); err != nil {
			return nil, fmt.Errorf("runlet/parser: strip shebang: %w", err)
		}
		s.HadShebang = true
	}

	rest, err := io.ReadAll(br)
	if err != nil {
		return nil, fmt.Errorf("runlet/parser: read body: %w", err)
	}
	s.Source = rest

	for _, line := range strings.Split(string(rest), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, magicPrefix) {
			continue
		}
		// Ensure the prefix is followed by whitespace or end-of-line,
		// otherwise "// runlet:depending" would be misclassified.
		after := trimmed[len(magicPrefix):]
		if after != "" && after[0] != ' ' && after[0] != '\t' {
			continue
		}
		payload := strings.TrimSpace(after)
		if payload == "" {
			return nil, fmt.Errorf("runlet/parser: bare %q comment has no fields", magicPrefix)
		}
		dep, derr := parseDep(payload)
		if derr != nil {
			return nil, fmt.Errorf("runlet/parser: %w", derr)
		}
		s.Deps = append(s.Deps, dep)
	}
	return s, nil
}

// looksLikeShebangSecondForm peeks deeper into the buffered reader to
// distinguish a real shebang trick (`//usr/bin/env ...`) from a
// regular Go doc comment. We do NOT advance the reader; the caller
// will read+discard the line if this returns true.
func looksLikeShebangSecondForm(br *bufio.Reader) bool {
	buf, err := br.Peek(16) // "//usr/bin/env "  → 14 chars
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	return strings.HasPrefix(string(buf), "//usr/bin/env ")
}

func parseDep(payload string) (Dep, error) {
	fields := strings.Fields(payload)
	if len(fields) != 2 {
		return Dep{}, fmt.Errorf("dep comment %q: want \"<module> <version>\", got %d fields", payload, len(fields))
	}
	if fields[0] == "" || fields[1] == "" {
		return Dep{}, fmt.Errorf("dep comment %q: empty field", payload)
	}
	return Dep{Path: fields[0], Version: fields[1]}, nil
}
