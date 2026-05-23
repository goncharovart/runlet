# CLAUDE.md — runlet

> Project context for AI coding agents. Keep in sync with reality.

## What this is

`runlet` — **single-binary Go script runner with PEP 723-style inline dependencies**. Write a `.go` file that declares its own dependencies inline; `runlet` resolves a transient module cache, compiles, and runs it. The Python equivalent is `uv run script.py`; the Go ecosystem had no answer until this.

Niche: Go has `go run`, but `go run` requires `go.mod` + neighbours. For one-off scripts (cron job utility, dev shell command, ops snippet), the ergonomics suck — you create a directory, init a module, `go mod tidy`, then `go run`. `runlet` collapses that to `runlet script.go`.

## Stack

- **Go 1.22+** (generics, modern atomic types)
- Standard library only for runtime (`os/exec`, `go/parser`, `crypto/sha256`, `path/filepath`)
- Module-aware build via `go build` invoked from cached scratch dirs
- No external runtime dependencies; vendored tools only in dev

## Project layout

```
runlet/
├── cmd/runlet/          # Main binary entry point
│   ├── main.go          # CLI dispatch
│   ├── cache.go         # `runlet cache info / clear` subcommands
│   ├── version.go       # `--version` via debug.ReadBuildInfo
│   └── *_test.go        # unit tests per file
├── internal/cachemgr/   # Module cache layout, locking, eviction
├── internal/parser/     # PEP 723-style header parser ( /// runlet ... /// )
├── internal/runner/     # Build + exec orchestration; platform.go for OS-specific
├── _examples/           # Runnable sample scripts (underscore-prefixed = not a Go package)
├── go.mod               # module github.com/goncharovart/runlet
├── README.md / README.ru.md / CHANGELOG.md
```

`cmd/`+`internal/` layout follows community-de-facto golang-standards/project-layout. Public packages — **none** (runlet is a binary, not a library); everything under `internal/` is private by Go's import rules.

## Build & test

```bash
# All tests with race + cover
go test -race -count=1 -cover ./...

# Single subdir
go test -race ./internal/parser/...

# Build binary
go build -o runlet ./cmd/runlet

# Install locally
go install ./cmd/runlet

# Vet + format
go vet ./...
gofmt -s -w .
goimports -w .

# Module hygiene
go mod tidy
```

## Coding conventions

### Idiomatic Go

- **Errors wrap context.** `fmt.Errorf("parse header in %s: %w", path, err)`.
- **Accept interfaces, return structs.** `parser.Result` is a struct; `runner.Runner` interface allows mocking in tests.
- **Context propagation.** `runner.Run(ctx, ...)` — all long-running operations accept a context first; cancellation cleans up build dirs.
- **Tests describe behavior in subtest names.** `t.Run("header without dependencies returns empty module list", ...)`.

### Script header parsing

The PEP 723-style header looks like:

```go
// /// runlet
// requires-go = "1.22"
// dependencies = [
//   "github.com/spf13/cobra v1.8.0",
// ]
// ///
package main
// ...
```

The parser:
- Reads until first non-comment line (the `package` declaration)
- Validates that header is **at top of file** (no blank line above first `///` marker)
- Parses inline TOML between `/// runlet` and `///`
- Returns a structured `Result` with `RequiresGo` (semver) + `Dependencies` slice

When extending the header format, **update both `parser/parser.go` and `_examples/`** so the docs stay testable.

### Cache layout

```
$XDG_CACHE_HOME/runlet/  (or %LOCALAPPDATA%\runlet\Cache\ on Windows)
├── <sha256(header+source)>/
│   ├── go.mod
│   ├── main.go
│   └── runlet.bin       # cached binary (if --persist-binary)
└── lock                 # file-lock for concurrent runlet invocations
```

Eviction strategy: LRU on `accessed_at` mtime; threshold configurable via env `RUNLET_CACHE_MAX_MB`. Default 1 GiB.

### Testing

- **Table-driven tests with subtests.** Each scenario named by behavior, not by index.
- **`-race` clean.** All tests pass `-race -count=10` before commit.
- **Coverage target: 80%+** on `internal/` packages.
- **Platform-specific tests gated by build tags** (`//go:build windows`). Don't mix cross-platform paths in a single test file.

## Pre-commit hook (recommended)

```bash
#!/usr/bin/env bash
# .githooks/pre-commit
set -e
gofmt -s -w .
go vet ./...
go test -race -short -count=1 ./...
golangci-lint run --new-from-rev=HEAD~1 || true
```

Enable: `git config core.hooksPath .githooks`

## Stability

v0.1.x — public CLI flags stable; internal/ packages **explicitly unstable**, may rename/reshape between minor versions. Migration notes in CHANGELOG.

## Related docs

- `README.md` — user-facing CLI overview, install, usage examples
- `README.ru.md` — Russian translation
- `CHANGELOG.md` — Keep-a-Changelog format
- `_examples/` — runnable sample scripts demonstrating header syntax
- `CONTRIBUTING.md` — issue → discussion → PR workflow
