// Package cachemgr is the operational layer for the runlet cache —
// the user-facing `runlet cache info` and `runlet cache clear`
// subcommands that the runner's internal cache logic does not expose.
//
// The cache itself is owned by internal/runner and is keyed on a
// SHA-256 hash of (script source, sorted deps). cachemgr knows about
// that layout but does not duplicate the keying — it just enumerates
// directories and reports / wipes them.
package cachemgr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRoot picks the on-disk location for the runlet cache. The
// resolution order matches runner.resolveCacheRoot so a `cache info`
// reads exactly what `runlet hello.go` would have written.
func ResolveRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "runlet"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("runlet/cachemgr: resolve cache dir: %w", err)
	}
	return filepath.Join(dir, "runlet"), nil
}

// Info is the human-readable summary printed by `runlet cache info`.
type Info struct {
	Root    string
	Exists  bool
	Scripts int   // number of cached scripts (subdirectories)
	Bytes   int64 // total size on disk in bytes
}

// Inspect walks the cache root and returns a populated Info. If the
// directory does not exist, Exists is false and the other fields are
// zero — this is not an error.
func Inspect(root string) (*Info, error) {
	out := &Info{Root: root}

	st, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("runlet/cachemgr: stat root: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("runlet/cachemgr: %s is not a directory", root)
	}
	out.Exists = true

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("runlet/cachemgr: read root: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out.Scripts++
		if sz, derr := dirSize(filepath.Join(root, e.Name())); derr == nil {
			out.Bytes += sz
		}
	}
	return out, nil
}

// Clear wipes the cache directory and recreates it empty. Idempotent:
// removing a non-existent directory is not an error. The caller is
// expected to have already obtained confirmation.
func Clear(root string) error {
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("runlet/cachemgr: remove %s: %w", root, err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("runlet/cachemgr: recreate %s: %w", root, err)
	}
	return nil
}

// PrintInfo writes a one-screen text summary to w. Intentionally
// terse — no tables, no colour, nothing that would not survive being
// piped through `grep`. Two lines for an empty cache, three for a
// populated one.
func PrintInfo(w io.Writer, info *Info) error {
	if !info.Exists {
		_, err := fmt.Fprintf(w, "runlet cache: %s\n  (directory does not exist; cache is empty)\n", info.Root)
		return err
	}
	_, err := fmt.Fprintf(w,
		"runlet cache: %s\n  scripts: %d\n  size:    %s\n",
		info.Root, info.Scripts, humanBytes(info.Bytes),
	)
	return err
}

// dirSize sums the sizes of all regular files under p. Symlinks are
// followed only at the first level so we don't recurse out of the
// cache tree by accident.
func dirSize(p string) (int64, error) {
	var total int64
	err := filepath.Walk(p, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// humanBytes formats a byte count with a binary-prefix suffix. We
// don't pull in a units library for this — fifteen lines of code is
// less risk than one external dependency in a tool that runs other
// people's code.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	suffixes := "KMGTPE"
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
	}
	out := fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), suffixes[exp])
	return strings.ReplaceAll(out, ".0 ", " ")
}
