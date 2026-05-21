package runner

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goncharovart/runlet/internal/parser"
)

var _ = bytes.NewBuffer // keep import used after refactor

// TestRun_HelloWorldNoDeps exercises the full happy path with a
// dependency-free script: parse → materialise → `go run` → capture
// stdout. Requires a Go toolchain in PATH; skipped otherwise.
func TestRun_HelloWorldNoDeps(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go in PATH; skipping integration test")
	}

	src := `package main

import "fmt"

func main() { fmt.Println("hello-from-runlet") }
`
	script, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	exit, err := Run(Config{
		Script:    script,
		CacheRoot: t.TempDir(),
		Stdout:    &out,
		Stderr:    &errBuf,
	})
	if err != nil {
		t.Fatalf("Run: %v (stderr=%q)", err, errBuf.String())
	}
	if exit != 0 {
		t.Fatalf("exit code %d, stderr=%q", exit, errBuf.String())
	}
	if !strings.Contains(out.String(), "hello-from-runlet") {
		t.Errorf("stdout=%q does not contain hello-from-runlet", out.String())
	}
}

// TestRun_ScriptExitCodePropagated — non-zero exit from the child
// process surfaces as the (exit, nil) return value.
func TestRun_ScriptExitCodePropagated(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go in PATH; skipping integration test")
	}

	src := `package main

import "os"

func main() { os.Exit(42) }
`
	script, _ := parser.Parse(strings.NewReader(src))
	exit, err := Run(Config{Script: script, CacheRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("Run returned error %v; want exit propagated cleanly", err)
	}
	if exit != 42 {
		t.Errorf("exit = %d, want 42", exit)
	}
}

// TestCacheKey_DeterministicForReorderedDeps — two scripts with the
// same source and the same dep set (in different order) hash to the
// same cache key, so `runlet foo.go` does not redundantly tidy on
// reorder.
func TestCacheKey_DeterministicForReorderedDeps(t *testing.T) {
	a := &parser.Script{
		Source: []byte("package main"),
		Deps: []parser.Dep{
			{Path: "github.com/a/x", Version: "v1.0.0"},
			{Path: "github.com/b/y", Version: "v2.0.0"},
		},
	}
	b := &parser.Script{
		Source: []byte("package main"),
		Deps: []parser.Dep{
			{Path: "github.com/b/y", Version: "v2.0.0"},
			{Path: "github.com/a/x", Version: "v1.0.0"},
		},
	}
	if cacheKey(a) != cacheKey(b) {
		t.Errorf("reordering deps changed cacheKey: %q vs %q", cacheKey(a), cacheKey(b))
	}
}

func TestCacheKey_ChangesWhenSourceChanges(t *testing.T) {
	a := &parser.Script{Source: []byte("package main\n// v1"), Deps: nil}
	b := &parser.Script{Source: []byte("package main\n// v2"), Deps: nil}
	if cacheKey(a) == cacheKey(b) {
		t.Error("different source produced the same cache key")
	}
}

func TestWriteGoMod_RendersDeps(t *testing.T) {
	dir := t.TempDir()
	err := writeGoMod(dir, []parser.Dep{
		{Path: "github.com/spf13/cobra", Version: "v1.9.0"},
		{Path: "github.com/jackc/pgx/v5", Version: "v5.9.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := mustReadFile(t, filepath.Join(dir, "go.mod"))
	if !strings.Contains(body, "module runlet/script") {
		t.Errorf("go.mod missing module declaration: %q", body)
	}
	if !strings.Contains(body, "github.com/spf13/cobra v1.9.0") {
		t.Errorf("go.mod missing cobra dep: %q", body)
	}
	if !strings.Contains(body, "github.com/jackc/pgx/v5 v5.9.2") {
		t.Errorf("go.mod missing pgx dep: %q", body)
	}
}

func TestWriteGoMod_NoDepsOmitsRequireBlock(t *testing.T) {
	dir := t.TempDir()
	if err := writeGoMod(dir, nil); err != nil {
		t.Fatal(err)
	}
	body := mustReadFile(t, filepath.Join(dir, "go.mod"))
	if strings.Contains(body, "require") {
		t.Errorf("go.mod with no deps must not contain `require`: %q", body)
	}
}

func mustReadFile(t *testing.T, p string) string {
	t.Helper()
	b, err := readFileNative(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
