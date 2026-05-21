# Contributing to runlet

Thanks for considering a contribution. Bug reports, small feature
PRs, and additional `examples/` scripts are all welcome.

## Quick orientation

The codebase is intentionally tiny:

| File / dir | Purpose |
|------------|---------|
| `cmd/runlet/main.go` | CLI entry point — open file (or stdin), call parser → runner |
| `internal/parser/` | Magic-comment grammar; shebang stripping; malformed-comment detection |
| `internal/runner/` | Content-addressed cache, synthesised `go.mod`, `go build` + exec |
| `examples/` | Short scripts that double as integration smoke-tests for new contributors |

There are no third-party Go dependencies in runlet itself — only
stdlib. That is a deliberate choice (every dep is a future supply-chain
risk for a tool that runs other people's code) and it should stay
that way unless the change has a strong argument.

## Before you open a PR

1. **Open an issue first** for anything beyond a typo or a doc fix.
   Coordinating early keeps PRs short and avoids parallel work.
2. **One change per PR.** A bug fix and a refactor in the same PR
   makes review three times longer than it needs to be.
3. **Tests.** Every new behaviour needs a test. The bar is: "if the
   test were deleted, the next refactor could break the behaviour
   silently."
4. **`go fmt`, `go vet`, `golangci-lint run`.** CI runs all three on
   ubuntu + macOS + windows × Go 1.22 + 1.25.
5. **Commit message:** imperative present tense, ≤72-char subject
   line, body explains *why*. Match the existing `git log`.

## Running tests locally

```bash
go test ./...
```

The runner package has integration tests that shell out to your local
`go` toolchain. They auto-skip when `go` is not on `PATH`, so a
parser-only contributor doesn't need Go installed to run the parser
suite.

## Magic-comment grammar — stability

The `// runlet:dep` grammar is the *only* surface the
script-writers see. Before `v0.1.0` it can shift; after `v0.1.0`
breaking changes require a major bump (`v0.2.0` resets the contract,
`v1.0.0` freezes it). If your PR touches the grammar:

- Add a test fixture under `internal/parser/testdata/`.
- Update the grammar section of the main README.
- Note the change in `CHANGELOG.md` under "Changed".

## Adding an example

`examples/` doubles as integration smoke-tests for newcomers. To
add one:

1. Single `.go` file with `// runlet:dep` declarations.
2. Keep it under ~25 lines — examples are for showing the value
   prop, not for being mini-projects.
3. Add a one-line description to README's "Examples" section.
4. The script *must* run end-to-end with the current `runlet`
   binary; otherwise the CI matrix breaks on the next push.

## Reporting bugs

A minimal reproducing script attached to the issue is the most
valuable thing you can include. Failing that, please give:

- runlet version (`runlet --version` after the v0.1.0 release; the
  output of `git rev-parse HEAD` if you built from source)
- `go version`
- OS + arch
- The script as a code block
- What you expected vs what happened

## License

By contributing you agree your work is released under the project's
[MIT license](LICENSE).
