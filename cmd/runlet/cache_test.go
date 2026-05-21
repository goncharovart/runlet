package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCache_InfoEmpty(t *testing.T) {
	// We point at a path that almost certainly does not exist on a
	// fresh test box; runCache should produce a clean "directory
	// does not exist" report. Resolution falls through to the
	// real default cache dir (os.UserCacheDir), so we accept any
	// stdout content that mentions cache state.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var out, errBuf bytes.Buffer
	exit := runCache([]string{"info"}, strings.NewReader(""), &out, &errBuf)
	if exit != 0 {
		t.Fatalf("exit = %d (stderr=%q)", exit, errBuf.String())
	}
	if !strings.Contains(out.String(), "runlet cache") {
		t.Errorf("expected 'runlet cache' in stdout, got %q", out.String())
	}
}

func TestRunCache_UnknownSubcommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	exit := runCache([]string{"vacuum"}, strings.NewReader(""), &out, &errBuf)
	if exit == 0 {
		t.Errorf("unknown subcommand should not exit 0; got %d", exit)
	}
	if !strings.Contains(errBuf.String(), "unknown") {
		t.Errorf("stderr should mention 'unknown'; got %q", errBuf.String())
	}
}

func TestRunCache_NoSubcommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	exit := runCache(nil, strings.NewReader(""), &out, &errBuf)
	if exit != 2 {
		t.Errorf("missing subcommand should exit 2; got %d", exit)
	}
	if !strings.Contains(errBuf.String(), "subcommand required") {
		t.Errorf("stderr should hint at required subcommand; got %q", errBuf.String())
	}
}

func TestRunCache_ClearWithYesFlag(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var out, errBuf bytes.Buffer
	exit := runCache([]string{"clear", "--yes"}, strings.NewReader(""), &out, &errBuf)
	if exit != 0 {
		t.Fatalf("exit = %d (stderr=%q)", exit, errBuf.String())
	}
	if !strings.Contains(out.String(), "cache cleared") {
		t.Errorf("expected 'cache cleared' in stdout, got %q", out.String())
	}
}

func TestRunCache_ClearAbortsOnNo(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var out, errBuf bytes.Buffer
	exit := runCache([]string{"clear"}, strings.NewReader("n\n"), &out, &errBuf)
	if exit != 0 {
		t.Errorf("aborted clear should still exit 0; got %d", exit)
	}
	if !strings.Contains(out.String(), "aborted") {
		t.Errorf("expected 'aborted' in stdout, got %q", out.String())
	}
}

func TestRunCache_ClearProceedsOnY(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var out, errBuf bytes.Buffer
	exit := runCache([]string{"clear"}, strings.NewReader("y\n"), &out, &errBuf)
	if exit != 0 {
		t.Fatalf("exit = %d (stderr=%q)", exit, errBuf.String())
	}
	if !strings.Contains(out.String(), "cache cleared") {
		t.Errorf("expected 'cache cleared' in stdout, got %q", out.String())
	}
}
