// Package runner materialises a parsed runlet Script into a temporary
// Go module on disk and hands it off to the host `go` toolchain.
//
// The cache is content-addressed: a SHA-256 of the script source plus
// the sorted dep list is the directory name, so identical inputs share
// the same workspace and identical scripts skip the `go mod tidy`
// roundtrip on every run after the first.
package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goncharovart/runlet/internal/parser"
)

// Config controls a single run.
type Config struct {
	// Script is the parsed file ready to materialise.
	Script *parser.Script

	// ScriptPath is the original on-disk path of the source file. It
	// is used only for diagnostics (errors mention the source the user
	// supplied) and to derive the synthesised module name.
	ScriptPath string

	// CacheRoot overrides the default cache directory. Tests use this
	// to point at t.TempDir(); production leaves it empty so the
	// XDG-derived default kicks in.
	CacheRoot string

	// GoBin overrides the `go` binary that is shelled out to. Empty
	// means PATH lookup ("go").
	GoBin string

	// Args are the arguments forwarded to the script after `go run`.
	Args []string

	// Stdin / Stdout / Stderr wire the child process to the parent.
	// When nil they default to os.Stdin/os.Stdout/os.Stderr.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run materialises the script under CacheRoot, writes a synthesised
// go.mod, then shells out to `go run`. It returns the child's exit
// code (0 on success) and a non-nil error only when materialisation
// itself fails — a non-zero exit from the script is reflected in the
// return value, not the error.
func Run(cfg Config) (int, error) {
	if cfg.Script == nil {
		return 0, fmt.Errorf("runlet/runner: Script is nil")
	}
	if cfg.GoBin == "" {
		cfg.GoBin = "go"
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}

	root, err := resolveCacheRoot(cfg.CacheRoot)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return 0, fmt.Errorf("runlet/runner: mkdir cache root: %w", err)
	}

	key := cacheKey(cfg.Script)
	workdir := filepath.Join(root, key)
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return 0, fmt.Errorf("runlet/runner: mkdir workdir: %w", err)
	}

	if err := writeGoMod(workdir, cfg.Script.Deps); err != nil {
		return 0, err
	}

	scriptName := "main.go"
	if cfg.ScriptPath != "" {
		base := filepath.Base(cfg.ScriptPath)
		if strings.HasSuffix(base, ".go") {
			scriptName = base
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, scriptName), cfg.Script.Source, 0o644); err != nil {
		return 0, fmt.Errorf("runlet/runner: write source: %w", err)
	}

	// `go mod tidy` resolves and downloads the declared deps. If the
	// cached workdir already has a populated go.sum, tidy is fast
	// (no network) and idempotent — that is the cache-hit path.
	if len(cfg.Script.Deps) > 0 {
		tidy := exec.Command(cfg.GoBin, "mod", "tidy")
		tidy.Dir = workdir
		tidy.Stdout = cfg.Stderr // mute on stderr so tidy noise does not pollute the script's stdout
		tidy.Stderr = cfg.Stderr
		if err := tidy.Run(); err != nil {
			return 0, fmt.Errorf("runlet/runner: go mod tidy: %w", err)
		}
	}

	// Build to a cached binary and exec it directly. We don't use
	// `go run` because it wraps the script's non-zero exit code in
	// its own exit-1 + "exit status N" stderr message, which means
	// callers cannot distinguish a script error from a build error.
	binName := "runlet-script.bin"
	if runtimeIsWindows() {
		binName = "runlet-script.exe"
	}
	binPath := filepath.Join(workdir, binName)
	build := exec.Command(cfg.GoBin, "build", "-o", binName, ".")
	build.Dir = workdir
	build.Stdout = cfg.Stderr
	build.Stderr = cfg.Stderr
	if err := build.Run(); err != nil {
		return 0, fmt.Errorf("runlet/runner: go build: %w", err)
	}

	cmd := exec.Command(binPath, cfg.Args...)
	cmd.Dir = workdir
	cmd.Stdin = cfg.Stdin
	cmd.Stdout = cfg.Stdout
	cmd.Stderr = cfg.Stderr

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 0, fmt.Errorf("runlet/runner: exec script: %w", err)
	}
	return 0, nil
}

// resolveCacheRoot picks the on-disk location for the runlet cache.
// CacheRoot in Config wins; otherwise XDG/LocalAppData defaults apply.
func resolveCacheRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "runlet"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("runlet/runner: resolve cache dir: %w", err)
	}
	return filepath.Join(dir, "runlet"), nil
}

// cacheKey produces a stable directory name for the given script.
// Same source + same deps → same key, so the second run is a fast hit.
func cacheKey(s *parser.Script) string {
	h := sha256.New()
	h.Write(s.Source)

	// Sort deps so reordering magic comments doesn't invalidate the
	// cache — semantically the module set is the same.
	deps := make([]parser.Dep, len(s.Deps))
	copy(deps, s.Deps)
	sort.Slice(deps, func(i, j int) bool { return deps[i].Path < deps[j].Path })

	for _, d := range deps {
		fmt.Fprintf(h, "\n%s@%s", d.Path, d.Version)
	}
	return hex.EncodeToString(h.Sum(nil)[:16]) // first 128 bits — plenty for cache
}

// writeGoMod synthesises a go.mod for the script. Module name is
// derived from the cache key so two scripts never share a name even
// when both are materialised in nested temp dirs (which doesn't
// happen in production but does in some test layouts).
func writeGoMod(workdir string, deps []parser.Dep) error {
	var sb strings.Builder
	sb.WriteString("module runlet/script\n\n")
	sb.WriteString("go 1.22\n")
	if len(deps) > 0 {
		sb.WriteString("\nrequire (\n")
		for _, d := range deps {
			fmt.Fprintf(&sb, "\t%s %s\n", d.Path, d.Version)
		}
		sb.WriteString(")\n")
	}
	if err := os.WriteFile(filepath.Join(workdir, "go.mod"), []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("runlet/runner: write go.mod: %w", err)
	}
	return nil
}
