# runlet

> Single-binary runner для Go-скриптов с inline-зависимостями.
> Как `uv run` для Python или `cargo script` для Rust — только для Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/goncharovart/runlet.svg)](https://pkg.go.dev/github.com/goncharovart/runlet)
[![Go Report Card](https://goreportcard.com/badge/github.com/goncharovart/runlet)](https://goreportcard.com/report/github.com/goncharovart/runlet)
[![CI](https://github.com/goncharovart/runlet/actions/workflows/ci.yml/badge.svg)](https://github.com/goncharovart/runlet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/goncharovart/runlet?sort=semver&display_name=tag&color=blue)](https://github.com/goncharovart/runlet/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![README (English)](https://img.shields.io/badge/README-English-blue.svg)](README.md)

> ⚠️ **Early.** `v0.1.0` зафиксирован, но грамматика `// runlet:dep` и CLI-поверхность остаются нестабильными до `v1.0.0` — breaking changes между минорными версиями допустимы и будут отмечены в [CHANGELOG.md](CHANGELOG.md).

---

Напиши один `.go` файл, задекларируй его зависимости в magic-comment, запусти —
без `go.mod`, без `go.sum`, без церемонии `mkdir mytool && go mod init`.

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

Запуск:

```bash
runlet hello.go
```

Всё. `runlet` парсит dependency-комменты, синтезирует throwaway `go.mod` в
content-addressed кэше, скачивает зависимости и передаёт файл в `go build` +
exec напрямую (не `go run`).

## Зачем это нужно

Другие языки закрыли этот gap некоторое время назад:

| | Inline-deps runner |
|---|---|
| **Python** | `uv run` (PEP 723 с `uv` 0.4) — first-class |
| **Rust** | `cargo +nightly script` ([RFC 3424](https://rust-lang.github.io/rfcs/3424-cargo-script.html)) — стабилизируется |
| **JS/TS** | `bunx`, `deno run` — встроено в runtime |
| **Go** | — |

`go run` работает с `.go` файлами но резолвит только stdlib. Что-то за пределами
требует полного module layout. Для одноразовых скриптов — "распарси этот лог,
запроси этот endpoint, отформатируй результат" — module-церемония это и есть
то, что убивает workflow.

`gorun` ([erning/gorun](https://github.com/erning/gorun), 2013) был первой
попыткой. Он предшествует Go modules и не обрабатывает inline-декларации
зависимостей.

`runlet` — современный взгляд: PEP 723-style inline deps, Go modules под
капотом, content-addressed кэш так что второй запуск того же скрипта
стартует за миллисекунды.

## Установка

```bash
go install github.com/goncharovart/runlet/cmd/runlet@latest
```

Требуется Go 1.22+. Скачанный бинарь `runlet` — единственное предварительное
требование, он shell-out к существующему `go` toolchain.

## Грамматика magic-комментариев

Скрипт декларирует зависимости через линейные комментарии в начале файла:

```
// runlet:dep <module path> <version>
```

- Комментарии где угодно в файле допустимы, но конвенционально в начале
- `<version>` следует Go module versioning: semver tag (`v1.2.3`),
  pseudo-version (`v0.0.0-20240101120000-abcdef012345`), или `latest`
  (резолвится при первом запуске, потом замораживается в кэше)
- Множественные `runlet:dep` строки допустимы; каждая становится `require`
  директивой в синтезированном `go.mod`

## Shebang

Сделай скрипт напрямую исполняемым:

```go
//usr/bin/env runlet "$0" "$@"; exit
// runlet:dep github.com/spf13/cobra v1.9.0

package main
// ...
```

Трюк `//usr/bin/env ...; exit` — стандартный Go-workaround для того, что
настоящие `#!` shebang ломают Go parser. `runlet` strip-ает строку перед
передачей файла в `go run`.

Сделай исполняемым через `chmod +x script.go` и вызывай как `./script.go`.

## Кэш

`runlet` кеширует синтезированные модули в `$XDG_CACHE_HOME/runlet/` (или
`~/.cache/runlet/` на Linux/macOS, `%LOCALAPPDATA%\runlet\` на Windows).
Cache key — content hash скрипта + сортированного списка deps, так что:

- Изменение скрипта инвалидирует только запись этого скрипта в кэше
- Два скрипта с одинаковыми deps переиспользуют одни и те же скачанные модули
  (через стандартный Go module cache `GOMODCACHE`)

Инспекция и очистка:

```bash
runlet cache info    # размер кэша + количество скриптов
runlet cache clear   # стереть кэш (с подтверждением)
runlet cache clear --yes   # без подтверждения
```

## Версия

```bash
runlet --version
# runlet v0.1.0 (abcdef012345) built 2026-05-21T19:00:00Z · go1.25.0
```

Использует `runtime/debug.ReadBuildInfo()` — никаких `-ldflags -X` при сборке
не требуется.

## Статус

`v0.1.0` зафиксирован. В коробке: CLI, parser, runner с content-addressed
кэшем, два формата shebang, example-скрипт, и Go 1.22+1.25 × ubuntu/macOS/windows
CI matrix.

Roadmap к `v0.2.0`:

- [x] CLI scaffold, parser, basic runner
- [x] Cache layer (SHA-256 content-addressed, sorted-dep stable)
- [x] Два формата shebang strip
- [x] First example bundle (`_examples/hello-lipgloss.go`)
- [x] CI с `-race`, `golangci-lint`, three-OS matrix
- [x] [#1](https://github.com/goncharovart/runlet/issues/1) `runlet cache info` / `cache clear`
- [x] [#3](https://github.com/goncharovart/runlet/issues/3) `runlet --version` через `debug.ReadBuildInfo`
- [ ] [#2](https://github.com/goncharovart/runlet/issues/2) Multi-file scripts (`runlet ./scripts/`)
- [ ] [#4](https://github.com/goncharovart/runlet/issues/4) Pre-run trust signal на `runlet:dep` modules
- [ ] [#5](https://github.com/goncharovart/runlet/issues/5) Homebrew tap + apt repo

Открой issue или discussion — дизайн-фидбэк на грамматику magic-комментариев
особенно welcome до того как `v1.0.0` её заморозит.

## Лицензия

MIT. См. [LICENSE](LICENSE).
