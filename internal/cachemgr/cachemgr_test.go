package cachemgr

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspect_NonExistentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	got, err := Inspect(root)
	if err != nil {
		t.Fatalf("Inspect on missing dir should not error: %v", err)
	}
	if got.Exists {
		t.Error("Exists should be false for missing dir")
	}
	if got.Scripts != 0 || got.Bytes != 0 {
		t.Errorf("expected zero counts, got Scripts=%d Bytes=%d", got.Scripts, got.Bytes)
	}
}

func TestInspect_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Exists {
		t.Error("Exists should be true for present empty dir")
	}
	if got.Scripts != 0 {
		t.Errorf("Scripts = %d, want 0", got.Scripts)
	}
}

func TestInspect_PopulatedRoot(t *testing.T) {
	root := t.TempDir()
	// Two "scripts" with content.
	for _, sub := range []string{"abc123", "def456"} {
		dir := filepath.Join(root, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), bytes.Repeat([]byte("a"), 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// And a stray file at the root — should NOT count as a script.
	if err := os.WriteFile(filepath.Join(root, "stray.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scripts != 2 {
		t.Errorf("Scripts = %d, want 2", got.Scripts)
	}
	if got.Bytes < 2048 {
		t.Errorf("Bytes = %d, want at least 2048 (2x1KiB files)", got.Bytes)
	}
}

func TestClear_RemovesAndRecreates(t *testing.T) {
	root := t.TempDir()
	junkDir := filepath.Join(root, "abc123")
	if err := os.MkdirAll(junkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(junkDir, "f.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Clear(root); err != nil {
		t.Fatal(err)
	}

	st, err := os.Stat(root)
	if err != nil {
		t.Fatalf("root should still exist after Clear: %v", err)
	}
	if !st.IsDir() {
		t.Error("root should be a directory")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Errorf("Clear left entries behind: %v", entries)
	}
}

func TestClear_OnMissingRootIsNoError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "never-existed")
	if err := Clear(root); err != nil {
		t.Fatalf("Clear on missing dir errored: %v", err)
	}
	// And after Clear the directory exists (recreated empty).
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Clear did not recreate root: %v", err)
	}
}

func TestPrintInfo_EmptyCache(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintInfo(&buf, &Info{Root: "/tmp/nope", Exists: false}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "does not exist") {
		t.Errorf("PrintInfo missing 'does not exist' for empty cache: %q", buf.String())
	}
}

func TestPrintInfo_PopulatedCache(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintInfo(&buf, &Info{Root: "/tmp/r", Exists: true, Scripts: 3, Bytes: 4096}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"/tmp/r", "scripts: 3", "size:"} {
		if !strings.Contains(out, want) {
			t.Errorf("PrintInfo output missing %q in %q", want, out)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1 MiB"},
		{1024 * 1024 * 1024, "1 GiB"},
	}
	for _, c := range cases {
		got := humanBytes(c.in)
		if got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveRoot_ExplicitWins(t *testing.T) {
	got, err := ResolveRoot("/custom/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/path" {
		t.Errorf("explicit override ignored: %q", got)
	}
}

func TestResolveRoot_XDGEnv(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
	got, err := ResolveRoot("")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/xdg/cache", "runlet") {
		t.Errorf("XDG path not honored: %q", got)
	}
}
