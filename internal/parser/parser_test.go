package parser

import (
	"strings"
	"testing"
)

func TestParse_NoDeps(t *testing.T) {
	src := `package main

import "fmt"

func main() { fmt.Println("hi") }
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(s.Deps))
	}
	if s.HadShebang {
		t.Error("plain Go file: HadShebang should be false")
	}
	if string(s.Source) != src {
		t.Error("Source should be returned unchanged when there is no shebang")
	}
}

func TestParse_OneDep(t *testing.T) {
	src := `// runlet:dep github.com/spf13/cobra v1.9.0
package main

func main() {}
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(s.Deps))
	}
	if s.Deps[0].Path != "github.com/spf13/cobra" {
		t.Errorf("Path = %q, want github.com/spf13/cobra", s.Deps[0].Path)
	}
	if s.Deps[0].Version != "v1.9.0" {
		t.Errorf("Version = %q, want v1.9.0", s.Deps[0].Version)
	}
}

func TestParse_MultipleDepsPreserveOrder(t *testing.T) {
	src := `// runlet:dep github.com/spf13/cobra v1.9.0
// runlet:dep github.com/jackc/pgx/v5 v5.9.2
// runlet:dep golang.org/x/sync v0.17.0

package main
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"github.com/spf13/cobra",
		"github.com/jackc/pgx/v5",
		"golang.org/x/sync",
	}
	if len(s.Deps) != len(want) {
		t.Fatalf("dep count = %d, want %d", len(s.Deps), len(want))
	}
	for i, d := range s.Deps {
		if d.Path != want[i] {
			t.Errorf("dep[%d].Path = %q, want %q", i, d.Path, want[i])
		}
	}
}

func TestParse_HashbangShebangIsStripped(t *testing.T) {
	src := `#!/usr/bin/env runlet
// runlet:dep github.com/x/y v1.0.0
package main

func main() {}
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if !s.HadShebang {
		t.Error("HadShebang should be true for #! line")
	}
	if strings.Contains(string(s.Source), "#!") {
		t.Errorf("Source still contains shebang: %q", s.Source)
	}
	if len(s.Deps) != 1 {
		t.Errorf("dep count = %d, want 1", len(s.Deps))
	}
}

func TestParse_GoFriendlyShebangIsStripped(t *testing.T) {
	src := `//usr/bin/env runlet "$0" "$@"; exit
// runlet:dep github.com/x/y v1.0.0
package main

func main() {}
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if !s.HadShebang {
		t.Error("HadShebang should be true for //usr/bin/env trick")
	}
	if strings.HasPrefix(string(s.Source), "//usr/bin/env") {
		t.Errorf("Source still contains shebang trick: %q", s.Source[:40])
	}
}

func TestParse_RegularDoubleSlashCommentIsNotShebang(t *testing.T) {
	src := `// Package main does X.
package main

func main() {}
`
	s, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if s.HadShebang {
		t.Error("doc comment must not be treated as a shebang")
	}
	if string(s.Source) != src {
		t.Error("doc-comment files must be returned unchanged")
	}
}

func TestParse_MalformedDepIsError(t *testing.T) {
	cases := []string{
		`// runlet:dep github.com/x/y
package main`,
		`// runlet:dep
package main`,
		`// runlet:dep github.com/x/y v1.0.0 extra
package main`,
	}
	for i, src := range cases {
		_, err := Parse(strings.NewReader(src))
		if err == nil {
			t.Errorf("case %d: expected error for %q, got nil", i, src)
		}
	}
}
