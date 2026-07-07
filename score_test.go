package hotspot

import "testing"

func TestScoreRanksChurnTimesComplexity(t *testing.T) {
	// A: high churn, high complexity -> should rank first.
	// B: high churn, trivial          -> low score.
	// C: dormant but very complex      -> low score.
	// D: nothing                       -> zero.
	files := []FileRisk{
		{Path: "b.go", Commits: 40, Complexity: 2},
		{Path: "a.go", Commits: 40, Complexity: 80},
		{Path: "c.go", Commits: 1, Complexity: 80},
		{Path: "d.go", Commits: 0, Complexity: 0},
	}
	Score(files)

	if files[0].Path != "a.go" {
		t.Fatalf("expected a.go first, got %q (order: %v)", files[0].Path, order(files))
	}
	if got := files[len(files)-1]; got.Path != "d.go" || got.Score != 0 {
		t.Fatalf("expected d.go last with score 0, got %q score %v", got.Path, got.Score)
	}
	// The churny-trivial and dormant-complex files must both score below the
	// genuine hotspot.
	for _, f := range files {
		if f.Path == "a.go" {
			continue
		}
		if f.Score >= scoreOf(files, "a.go") {
			t.Fatalf("%s (%.3f) should score below a.go (%.3f)", f.Path, f.Score, scoreOf(files, "a.go"))
		}
	}
}

func TestScoreNormalizedToUnitInterval(t *testing.T) {
	files := []FileRisk{
		{Path: "x", Commits: 100, Complexity: 100},
		{Path: "y", Commits: 3, Complexity: 9},
	}
	Score(files)
	for _, f := range files {
		if f.ChurnNorm < 0 || f.ChurnNorm > 1 || f.ComplexityNorm < 0 || f.ComplexityNorm > 1 {
			t.Fatalf("%s norms out of range: churn=%v cmp=%v", f.Path, f.ChurnNorm, f.ComplexityNorm)
		}
		if f.Score < 0 || f.Score > 1 {
			t.Fatalf("%s score out of range: %v", f.Path, f.Score)
		}
	}
	// The max on both axes must reach 1.0 and therefore score 1.0.
	if files[0].Score != 1 {
		t.Fatalf("top file should score 1.0, got %v", files[0].Score)
	}
}

func TestScoreEmptyAndAllZero(t *testing.T) {
	Score(nil) // must not panic
	files := []FileRisk{{Path: "a"}, {Path: "b"}}
	Score(files) // all-zero: no divide-by-zero, all scores 0
	for _, f := range files {
		if f.Score != 0 {
			t.Fatalf("all-zero input should yield score 0, got %v for %s", f.Score, f.Path)
		}
	}
}

func order(files []FileRisk) []string {
	s := make([]string, len(files))
	for i, f := range files {
		s[i] = f.Path
	}
	return s
}

func scoreOf(files []FileRisk, path string) float64 {
	for _, f := range files {
		if f.Path == path {
			return f.Score
		}
	}
	return -1
}
