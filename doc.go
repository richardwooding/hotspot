// Package hotspot ranks a codebase's refactoring risk by fusing signals that,
// on their own, each tell only half the story.
//
// The core insight (behavioral code analysis): the code most worth your
// attention is code that is both hard to understand *and* changes often —
// high complexity on low-churn code is dormant, and high churn on trivial code
// is cheap. Multiplying the two surfaces the genuine hotspots.
//
// hotspot composes existing single-purpose libraries rather than reimplementing
// them:
//
//   - churn      github.com/richardwooding/gitmeta      (per-file commit count)
//   - complexity github.com/richardwooding/codemetrics  (cyclomatic + cognitive)
//
// Each signal is collected behind a small interface (see [ChurnProvider] and
// [ComplexityProvider]) so further dimensions — package coupling
// (go-coupling) and call-graph centrality (treesitter-symbols) — drop in
// without touching the scorer.
//
// The library entry point is [Analyze]; [Score] is the pure, dependency-free
// ranking step and is exported for testing and reuse.
package hotspot
