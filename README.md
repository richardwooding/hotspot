# hotspot

[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/hotspot.svg)](https://pkg.go.dev/github.com/richardwooding/hotspot)
[![ci](https://github.com/richardwooding/hotspot/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/hotspot/actions/workflows/ci.yml)

**Website:** [richardwooding.github.io/hotspot](https://richardwooding.github.io/hotspot/)

Find the code **worth refactoring first**. `hotspot` ranks a repository by
refactoring risk using *behavioral code analysis*: the code that costs you is
the code that is both **complex** and **frequently changed**. Complexity that
never changes is dormant; churn on trivial code is cheap. Multiply the two and
the real hotspots rise to the top.

It's built by **composing** small single-purpose libraries rather than
reimplementing them:

| signal | source |
| --- | --- |
| **churn** — commits per file, last author/time | [`gitmeta`](https://github.com/richardwooding/gitmeta) |
| **complexity** — cyclomatic + cognitive | [`codemetrics`](https://github.com/richardwooding/codemetrics) |

## Install

```sh
go install github.com/richardwooding/hotspot/cmd/hotspot@latest
```

## Use

```sh
hotspot -top 20 /path/to/repo
```

```
SCORE  CHURN  COGN  CYC  FNS  LANG  FILE
0.912  6      68    47   10   go    detect.go
0.774  4      76    72   17   go    projectdetect_test.go
0.623  4      32    42   9    go    resolver_test.go
0.612  4      30    28   8    go    resolver.go
0.485  2      53    47   11   go    projectdetect.go
```

Flags: `-top N` (0 = all), `-min-score F`, `-json`, `-include-untracked`.

`-json` emits the full report — `score`, the normalized `churnNorm` /
`complexityNorm`, `commits`, `cognitive`, `cyclomatic`, `lastAuthor` — for
dashboards, bots, or a CI gate.

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

Each file's churn and complexity are passed through `log1p` (both are
heavy-tailed — one 500-commit file or one enormous function shouldn't flatten
everything else), min-max normalized to `0..1`, and multiplied:

```
score = normalized(log1p(commits)) × normalized(log1p(complexity))
```

A file scores high only when **both** signals are high. Missing either — a
dormant-but-complex file, or a churny-but-trivial one — pulls the score toward
zero, which is the point.

`hotspot` walks tracked source (skipping `.git`, `vendor`, `node_modules`,
build dirs); outside a git tree it degrades gracefully to a pure complexity
ranking.

## Status & roadmap

**v0** ranks **churn × complexity**. Complexity is computed for **Go** today
(via `codemetrics`' `go/ast` library path); other languages are walked but not
yet scored. Each signal sits behind a small interface (`ChurnProvider`,
`ComplexityProvider`), so the following drop in without changing the scorer:

- **coupling / instability** — [`go-coupling`](https://github.com/richardwooding/go-coupling)
- **call-graph centrality** (blast radius) — [`treesitter-symbols`](https://github.com/richardwooding/treesitter-symbols)
- **SARIF** output for GitHub code scanning — [`go-sarif`](https://github.com/richardwooding/go-sarif)
- an **MCP server** so an AI agent can ask *"what's the riskiest code here, and why?"*
- multi-language complexity

## License

MIT © Richard Wooding
