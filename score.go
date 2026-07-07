package hotspot

import (
	"math"
	"sort"
)

// CouplingWeight bounds how much package coupling can amplify a file's base
// risk: the amplifier is 1 + CouplingWeight×couplingNorm, so at the default a
// maximally-coupled file's base risk is doubled. Coupling only ever amplifies —
// a file with no coupling keeps its churn×complexity risk unchanged.
const CouplingWeight = 1.0

// Score fills ChurnNorm, ComplexityNorm, CouplingNorm and Score on each file
// (in place) and sorts the slice by Score descending.
//
// Churn, complexity and coupling degree (Ca+Ce) are each heavy-tailed, so each
// is passed through log1p before min-max normalization to 0..1 — this keeps a
// single 500-commit file or one enormous function from flattening everything
// else. The base risk is the product of churn and complexity: a file matters
// only when it is *both* frequently changed and complex (dormant complexity and
// churny-but-trivial files are not hotspots). Coupling then amplifies that base
// — code that is hot, complex *and* entangled with many other packages is the
// riskiest to touch. Scores are renormalized so the top file is 1.0 and the
// whole report reads as a relative risk index in 0..1.
func Score(files []FileRisk) {
	var churnMax, cmpMax, coupMax float64
	for _, f := range files {
		churnMax = math.Max(churnMax, log1p(float64(f.Commits)))
		cmpMax = math.Max(cmpMax, log1p(f.Complexity))
		coupMax = math.Max(coupMax, log1p(float64(f.Afferent+f.Efferent)))
	}
	raw := make([]float64, len(files))
	var rawMax float64
	for i := range files {
		files[i].ChurnNorm = norm(log1p(float64(files[i].Commits)), churnMax)
		files[i].ComplexityNorm = norm(log1p(files[i].Complexity), cmpMax)
		files[i].CouplingNorm = norm(log1p(float64(files[i].Afferent+files[i].Efferent)), coupMax)
		base := files[i].ChurnNorm * files[i].ComplexityNorm
		raw[i] = base * (1 + CouplingWeight*files[i].CouplingNorm)
		rawMax = math.Max(rawMax, raw[i])
	}
	for i := range files {
		files[i].Score = norm(raw[i], rawMax)
	}
	sort.SliceStable(files, func(a, b int) bool {
		return less(files[b], files[a]) // descending
	})
}

// less defines the total order used for ranking (ascending): Score, then
// Complexity, then Commits, then Path, so output is stable and deterministic.
func less(a, b FileRisk) bool {
	if a.Score != b.Score {
		return a.Score < b.Score
	}
	if a.Complexity != b.Complexity {
		return a.Complexity < b.Complexity
	}
	if a.Commits != b.Commits {
		return a.Commits < b.Commits
	}
	return a.Path > b.Path // lexically-earlier path ranks higher on full ties
}

func log1p(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log1p(x)
}

func norm(x, max float64) float64 {
	if max <= 0 {
		return 0
	}
	return x / max
}
