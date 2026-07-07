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

| signal | role | source |
| --- | --- | --- |
| **churn** — commits per file, last author/time | how often it changes | [`gitmeta`](https://github.com/richardwooding/gitmeta) |
| **complexity** — cyclomatic + cognitive | how hard it is | [`codemetrics`](https://github.com/richardwooding/codemetrics) |
| **coupling** — afferent/efferent (Ca/Ce) | how entangled it is | [`go-coupling`](https://github.com/richardwooding/go-coupling) |

## Install

```sh
go install github.com/richardwooding/hotspot/cmd/hotspot@latest
```

## Use

```sh
hotspot -top 20 /path/to/repo
```

```
SCORE  CHURN  COGN  CYC  CA  CE  FNS  LANG  FILE
1.000  6      68    47   1   1   10   go    detect.go
0.849  4      76    72   1   1   17   go    projectdetect_test.go
0.683  4      32    42   1   1   9    go    resolver_test.go
0.671  4      30    28   1   1   8    go    resolver.go
0.532  2      53    47   1   1   11   go    projectdetect.go
```

Flags: `-top N` (0 = all), `-min-score F`, `-json`, `-include-untracked`.

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

**Today** hotspot fuses **churn × complexity × coupling**. Complexity is
computed for **Go** (via `codemetrics`' `go/ast` library path) and coupling for
Go modules (via `go-coupling`); other languages are walked but not yet scored.
Each signal sits behind a small interface (`ChurnProvider`,
`ComplexityProvider`, `CouplingProvider`), so the following drop in without
changing the scorer:

- **call-graph centrality** (blast radius) — [`treesitter-symbols`](https://github.com/richardwooding/treesitter-symbols)
- **SARIF** output for GitHub code scanning — [`go-sarif`](https://github.com/richardwooding/go-sarif)
- an **MCP server** so an AI agent can ask *"what's the riskiest code here, and why?"*
- multi-language complexity & coupling

## License

MIT © Richard Wooding
