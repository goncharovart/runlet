package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionString_FullFacts(t *testing.T) {
	got := versionString(buildFacts{
		main:      "v0.1.0",
		revision:  "abcdef0123456789",
		buildTime: "2026-05-21T19:00:00Z",
		goVersion: "go1.25.0",
	})
	want := "runlet v0.1.0 (abcdef012345) built 2026-05-21T19:00:00Z · go1.25.0"
	if got != want {
		t.Errorf("versionString mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestVersionString_DirtyFlag(t *testing.T) {
	got := versionString(buildFacts{
		main:      "v0.1.0",
		revision:  "abcdef0123456789",
		dirty:     true,
		goVersion: "go1.25.0",
	})
	if !strings.Contains(got, "-dirty") {
		t.Errorf("versionString did not emit -dirty marker: %q", got)
	}
}

func TestVersionString_DevBuild(t *testing.T) {
	// Simulates the `go run cmd/runlet/main.go --version` case:
	// no module info, no VCS data, only the Go toolchain is known.
	got := versionString(buildFacts{
		main:      "(devel)",
		goVersion: "go1.22.0",
	})
	want := "runlet (devel) · go1.22.0"
	if got != want {
		t.Errorf("dev-build versionString\n got: %q\nwant: %q", got, want)
	}
}

func TestVersionString_TruncatesLongRevision(t *testing.T) {
	got := versionString(buildFacts{
		main:      "v0.1.0",
		revision:  "ffffffffffffffffffffffffffffffffffffffff", // 40-char SHA
		goVersion: "go1.25.0",
	})
	if !strings.Contains(got, "(ffffffffffff)") {
		t.Errorf("revision was not truncated to 12 chars: %q", got)
	}
	if strings.Contains(got, "(ffffffffffffffffffffffffffffffffffffffff") {
		t.Errorf("revision still present at full length: %q", got)
	}
}

// TestPrintVersion_WritesOneLine — integration with the real
// debug.ReadBuildInfo. We can't assert exact content (it depends on
// how the test binary was built), but the output must be one line,
// non-empty, and start with "runlet".
func TestPrintVersion_WritesOneLine(t *testing.T) {
	var buf bytes.Buffer
	if err := printVersion(&buf); err != nil {
		t.Fatalf("printVersion: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("printVersion produced empty output")
	}
	if !strings.HasPrefix(out, "runlet ") {
		t.Errorf("output should start with 'runlet '; got %q", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("expected exactly one newline; got %d in %q", strings.Count(out, "\n"), out)
	}
}

// TestCLI_VersionFlag — end-to-end through run(). Verifies both
// --version and -v exit cleanly with status 0.
func TestCLI_VersionFlag(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			// run() writes to os.Stdout/os.Stderr directly; we
			// can't easily intercept here without refactoring the
			// CLI. The interesting assertion is that the exit code
			// is 0 — i.e. the flag is recognised and routed to
			// printVersion rather than being mistaken for a script
			// path, which would fail with "open --version: ...".
			exit := run([]string{flag})
			if exit != 0 {
				t.Errorf("run([%q]) exit = %d, want 0", flag, exit)
			}
		})
	}
}
