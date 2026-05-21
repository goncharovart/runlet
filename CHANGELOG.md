# Changelog

All notable changes to runlet are tracked here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The public surface is the **`// runlet:dep` magic-comment grammar**
and the **`runlet` CLI**. Both are considered unstable until
`v1.0.0` — breaking changes between minor versions in the `v0.x`
series are allowed but will be called out under "Changed".

## [Unreleased]

Pending milestones live in the [GitHub issues](https://github.com/goncharovart/runlet/issues)
labelled `good first issue`.

## [v0.1.0] — 2026-05-21

First tagged release. The CLI, parser, runner, and content-addressed
cache all land together so v0.1.0 is "everything you need to write a
single-file Go script with third-party deps and run it with one
command."

### Added

- **CLI** — `runlet <script.go> [args...]`. The `-` form reads from
  stdin. Anything after the script path is forwarded as arguments
  to the script itself.
- **Magic-comment grammar** — `// runlet:dep <module path> <version>`.
  Module versions follow Go module syntax (`v1.2.3`, pseudo-versions,
  `latest`). Multiple `runlet:dep` comments compile into a single
  synthesised `require (...)` block.
- **Shebang support** — two forms are recognised and stripped before
  `go run`:
  - `#!/usr/bin/env runlet` (traditional shebang, breaks `go fmt`)
  - `//usr/bin/env runlet "$0" "$@"; exit` (Go-friendly trick — the
    line parses as a comment, so the file remains valid Go)
- **Content-addressed cache** — SHA-256 of `(source, sorted deps)` is
  the directory name under `$XDG_CACHE_HOME/runlet/`. Identical
  scripts reuse the same workspace and skip the `go mod tidy`
  roundtrip on subsequent runs.
- **`go build`-and-exec** instead of `go run`. This is the part that
  matters most for tooling: `go run` wraps a script's non-zero exit
  code into its own `exit-1 + "exit status N"` stderr, hiding the
  real code from callers. runlet builds to a cached binary and
  execs it directly, so `runlet myscript.go && echo ok` works the
  way you'd expect.
- **Examples** — `examples/hello-lipgloss.go` drives the value prop
  in 12 lines.

### Tests

13 unit tests covering:
- Parser: no-deps, one-dep, multi-dep ordering, both shebang forms,
  doc-comment-not-a-shebang, malformed comments.
- Runner: integration runs against a real `go` toolchain (auto-skipped
  when `go` is not on `PATH`), exit-code propagation, cache-key
  determinism, `go.mod` rendering with and without deps.

CI matrix: Go 1.22 and 1.25 × ubuntu-latest + macos-latest +
windows-latest.

### Install

```bash
go install github.com/goncharovart/runlet/cmd/runlet@v0.1.0
```

Requires Go 1.22+. The `runlet` binary is the only prerequisite —
it shells out to your existing `go` toolchain.

### Not yet

- `runlet cache info` / `runlet cache clear` subcommands (#1)
- Multi-file scripts — `runlet ./scripts/` (#2)
- `runlet --version` flag wired to `debug.ReadBuildInfo` (#3)
- pkg.go.dev–style trust signals on `runlet:dep` deps (#4)
- A real `runlet` Linux package (apt / brew tap) (#5)
