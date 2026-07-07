# hotspot

[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/hotspot.svg)](https://pkg.go.dev/github.com/richardwooding/hotspot)
[![ci](https://github.com/richardwooding/hotspot/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/hotspot/actions/workflows/ci.yml)

**Website:** [richardwooding.github.io/hotspot](https://richardwooding.github.io/hotspot/)

Find the code **worth refactoring first**, across **17 languages**. `hotspot`
ranks a repository by refactoring risk using *behavioral code analysis*: the
code that costs you is the code that is both **complex** and **frequently
changed**. Complexity that never changes is dormant; churn on trivial code is
cheap. Multiply the two and the real hotspots rise to the top.

It's built by **composing** small single-purpose libraries rather than
reimplementing them:

| signal | role | source |
| --- | --- | --- |
| **churn** — commits per file, last author/time | how often it changes | [`gitmeta`](https://github.com/richardwooding/gitmeta) |
| **complexity** — cyclomatic + cognitive, **17 languages** | how hard it is | [`codemetrics`](https://github.com/richardwooding/codemetrics) |
| **coupling** — afferent/efferent (Ca/Ce), **multi-language** | how entangled it is | [`go-coupling`](https://github.com/richardwooding/go-coupling) + [`treesitter-symbols`](https://github.com/richardwooding/treesitter-symbols) |

## Install

**Homebrew** (macOS):

```sh
brew install --cask richardwooding/tap/hotspot
```

**Go:**

```sh
go install github.com/richardwooding/hotspot/cmd/hotspot@latest
```

Or download a prebuilt binary for macOS, Linux or Windows from the
[releases page](https://github.com/richardwooding/hotspot/releases). It's a
single static binary — no cgo, no runtime dependencies (a `git` binary on PATH
is used for churn when present).

## CLI

Point it at any repository — the language is detected per file:

```sh
hotspot /path/to/repo          # top 20, ranked highest-risk first
hotspot -top 5 .               # just the 5 hottest files
hotspot -min-score 0.5 .       # only files at or above a risk index
hotspot -json . > hotspots.json
```

```
$ hotspot -top 6 .
SCORE  CHURN  COGN  CYC  CA  CE  FNS  LANG        FILE
1.000  34     57    24   6   3   12   python      sql/query.py
0.812  28     112   48   5   4   9    typescript  src/checker.ts
0.744  25     45    19   4   2   8    go          internal/completions.go
0.610  22     61    28   5   2   7    rust        src/ast.rs
0.470  15     43    21   3   2   6    php         src/Router.php
0.331  9      31    16   0   0   4    r           R/reshape.R
```

Columns: **SCORE** (0..1 relative risk index) · **CHURN** (commits) ·
**COGN** / **CYC** (cognitive / cyclomatic complexity) · **CA** / **CE**
(afferent / efferent coupling) · **FNS** (functions) · **LANG** · **FILE**.

| flag | default | meaning |
| --- | --- | --- |
| `-top N` | `20` | show the top N files (`0` = all) |
| `-min-score F` | `0` | hide files scoring below `F` (0..1) |
| `-json` | `false` | emit the full report as JSON |
| `-include-untracked` | `false` | also score files not tracked by git |
| `-version` | | print version and exit |

`-json` emits the full report — `score`, the normalized `churnNorm` /
`complexityNorm` / `couplingNorm`, `commits`, `cognitive`, `cyclomatic`,
`afferent`, `efferent`, `instability`, `lastAuthor` — for dashboards, bots, or
a CI gate.

## As a library

```go
rep, err := hotspot.Analyze(ctx, ".", hotspot.Options{})
for _, f := range rep.Files[:10] { // already sorted, highest risk first
    fmt.Printf("%.3f  %s\n", f.Score, f.Path)
}
```

`Score([]FileRisk)` is pure and exported, so you can rank pre-collected signals
without touching git or a parser.

## How the score works

Churn, complexity, and coupling degree (`Ca+Ce`) are each passed through
`log1p` (all heavy-tailed — one 500-commit file or one huge function shouldn't
flatten everything else) and min-max normalized to `0..1`. The base risk is
churn × complexity; coupling then **amplifies** it:

```
base  = normalized(log1p(commits)) × normalized(log1p(complexity))
score = base × (1 + normalized(log1p(Ca+Ce)))   // then renormalized so top = 1.0
```

A file scores high only when it is **both** frequently changed and complex —
missing either (dormant-but-complex, or churny-but-trivial) pulls it toward
zero. Coupling never zeroes a score; it only pushes hot, complex code that is
also **entangled with many packages** further up the list. Scores are
renormalized so the report reads as a relative risk index in `0..1`.

`hotspot` walks tracked source (skipping `.git`, `vendor`, `node_modules`,
build dirs); outside a git tree it degrades gracefully to a pure complexity
ranking.

## Status & roadmap

**Today** hotspot fuses **churn × complexity × coupling**, all multi-language:

- **Complexity** across **17 languages** — `c, cpp, csharp, go, java,
  javascript, kotlin, matlab, perl, php, python, r, ruby, rust, scala, swift,
  typescript` — with **Go** parsed via `go/ast` and the rest via `codemetrics`'
  **pure-Go tree-sitter** backend
  ([gotreesitter](https://github.com/odvcencio/gotreesitter), no cgo).
- **Coupling** across every ecosystem `go-coupling` understands — Go, Rust,
  Python, JS/TS, Ruby, C/C++, Swift, plus the declaration-based Java, Kotlin,
  Scala, C#, PHP and Perl. Imports are extracted per file by
  [`treesitter-symbols`](https://github.com/richardwooding/treesitter-symbols)
  (same pure-Go tree-sitter), fed to `go-coupling`, and its per-node Ca/Ce are
  joined back to each file via `go-coupling`'s `FileCoupling()`.
- **Churn** is language-agnostic.

Each signal sits behind a small interface (`ChurnProvider`,
`ComplexityProvider`, `CouplingProvider`), so the following drop in without
changing the scorer:

- **call-graph centrality** (blast radius) — `treesitter-symbols` already
  supplies the call edges
- **SARIF** output for GitHub code scanning — [`go-sarif`](https://github.com/richardwooding/go-sarif)
- an **MCP server** so an AI agent can ask *"what's the riskiest code here, and why?"*

## License

MIT © Richard Wooding
