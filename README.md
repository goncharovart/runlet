# runlet

> Single-binary runner for Go scripts with inline dependencies.
> Like `uv run` for Python or `cargo script` for Rust — but for Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/goncharovart/runlet.svg)](https://pkg.go.dev/github.com/goncharovart/runlet)
[![Go Report Card](https://goreportcard.com/badge/github.com/goncharovart/runlet)](https://goreportcard.com/report/github.com/goncharovart/runlet)
[![CI](https://github.com/goncharovart/runlet/actions/workflows/ci.yml/badge.svg)](https://github.com/goncharovart/runlet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/goncharovart/runlet?sort=semver&display_name=tag&color=blue)](https://github.com/goncharovart/runlet/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> ⚠️ **Early.** `v0.1.0` is tagged but the `// runlet:dep` grammar and CLI surface are unstable until `v1.0.0` — breaking changes between minor versions are allowed and will be called out in [CHANGELOG.md](CHANGELOG.md). Star/watch to follow.

---

Write a single `.go` file, declare its dependencies in a magic
comment, run it — no `go.mod`, no `go.sum`, no `mkdir mytool && go mod
init` ceremony.

```go
// runlet:dep github.com/charmbracelet/lipgloss v1.0.0
package main

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

func main() {
    style := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(lipgloss.Color("#7D56F4")).
        Padding(0, 1)
    fmt.Println(style.Render("hello from a single file"))
}
```

Run it:

```bash
runlet hello.go
```

That's it. `runlet` parses the dependency comments, synthesises a
throwaway `go.mod` in a content-addressed cache, downloads deps, and
hands the file to `go run`.

## Why this exists

Other languages closed this gap a while ago:

| | Inline-deps runner |
|---|---|
| **Python** | `uv run` (PEP 723 since `uv` 0.4) — first-class |
| **Rust** | `cargo +nightly script` ([RFC 3424](https://rust-lang.github.io/rfcs/3424-cargo-script.html)) — stabilising |
| **JS/TS** | `bunx`, `deno run` — built into the runtime |
| **Go** | — |

`go run` works on `.go` files but only resolves the stdlib. Anything
beyond that needs a full module layout. For one-off scripts —
"parse this log, query that endpoint, format the result" — the module
ceremony is the part that kills the workflow.

`gorun` ([erning/gorun](https://github.com/erning/gorun), 2013) was
the first attempt. It predates Go modules and does not handle inline
dependency declarations.

`runlet` is the modern take: PEP-723-style inline deps, Go-modules
under the hood, a content-addressed cache so the second run of the
same script starts in milliseconds.

## Install

```bash
go install github.com/goncharovart/runlet/cmd/runlet@latest
```

Requires Go 1.22+. The downloaded `runlet` binary is the only
prerequisite — it shells out to your existing `go` toolchain.

## Magic comment grammar

A script declares dependencies through line comments at the top of
the file:

```
// runlet:dep <module path> <version>
```

- Comments anywhere in the file are accepted, but conventionally
  they go at the top.
- `<version>` follows Go module versioning: a semver tag (`v1.2.3`),
  a pseudo-version (`v0.0.0-20240101120000-abcdef012345`), or
  `latest` (resolved at first run, then frozen in the cache).
- Multiple `runlet:dep` lines are allowed; each one becomes a
  `require` directive in the synthesised `go.mod`.

## Shebang

Make a script directly executable:

```go
//usr/bin/env runlet "$0" "$@"; exit
// runlet:dep github.com/spf13/cobra v1.9.0

package main
// ...
```

The `//usr/bin/env ...; exit` trick is the standard Go workaround
for the fact that real `#!` shebangs break Go's parser. `runlet`
strips the line before handing the file to `go run`.

Make it executable with `chmod +x script.go` and call it as
`./script.go`.

## Cache

`runlet` caches synthesised modules in `$XDG_CACHE_HOME/runlet/`
(or `~/.cache/runlet/` on Linux/macOS, `%LOCALAPPDATA%\runlet\` on
Windows). The cache key is a content hash of the script + sorted
dep list, so:

- Changing the script invalidates only that script's cache entry.
- Two scripts with identical deps reuse the same downloaded modules
  (through the standard Go module cache `GOMODCACHE`).
- `runlet cache clear` wipes everything.

## Status

`v0.1.0` is shipped. Inside the box: CLI, parser, runner with
content-addressed cache, two shebang formats, an example script, and
a Go-1.22+1.25 × ubuntu/macOS/windows CI matrix.

Roadmap to `v0.2.0`:

- [x] CLI scaffold, parser, basic runner
- [x] Cache layer (SHA-256 content-addressed, sorted-dep stable)
- [x] Two shebang formats stripped
- [x] First example bundle (`_examples/hello-lipgloss.go`)
- [x] CI with `-race`, `golangci-lint`, three-OS matrix
- [ ] [#1](https://github.com/goncharovart/runlet/issues/1) `runlet cache info` / `runlet cache clear`
- [ ] [#2](https://github.com/goncharovart/runlet/issues/2) Multi-file scripts (`runlet ./scripts/`)
- [ ] [#3](https://github.com/goncharovart/runlet/issues/3) `runlet --version` via `debug.ReadBuildInfo`
- [ ] [#4](https://github.com/goncharovart/runlet/issues/4) Pre-run trust signal on `runlet:dep` modules
- [ ] [#5](https://github.com/goncharovart/runlet/issues/5) Homebrew tap + apt repo distribution

Open an issue or a discussion — design feedback on the magic-comment
grammar is especially welcome before `v1.0.0` freezes it.

## License

MIT. See [LICENSE](LICENSE).
