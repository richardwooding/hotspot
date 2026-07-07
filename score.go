package hotspot

import (
	"math"
	"sort"
)

// Score fills ChurnNorm, ComplexityNorm and Score on each file (in place) and
// sorts the slice by Score descending.
//
// Both churn and complexity are heavy-tailed, so each is passed through
// log1p before min-max normalization to 0..1 — this keeps a single 500-commit
// file or one enormous function from flattening everything else to zero. The
// combined risk is the product of the two normalized signals: a file scores
// high only when it is *both* frequently changed and complex. A file missing
// either signal scores 0 on that axis and therefore 0 overall, which is the
// intended behavior — dormant complexity and churny-but-trivial files are not
// hotspots.
func Score(files []FileRisk) {
	var churnMax, cmpMax float64
	for _, f := range files {
		churnMax = math.Max(churnMax, log1p(float64(f.Commits)))
		cmpMax = math.Max(cmpMax, log1p(f.Complexity))
	}
	for i := range files {
		files[i].ChurnNorm = norm(log1p(float64(files[i].Commits)), churnMax)
		files[i].ComplexityNorm = norm(log1p(files[i].Complexity), cmpMax)
		files[i].Score = files[i].ChurnNorm * files[i].ComplexityNorm
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
