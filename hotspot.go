package hotspot

import (
	"context"
	"time"
)

// FileRisk is the per-file result: the raw signals, their normalized forms,
// and the combined risk Score. Files with no measurable complexity (e.g. an
// unsupported language) still carry churn and appear with a low score.
type FileRisk struct {
	Path       string    `json:"path"`       // repo-relative, slash-separated
	Language   string    `json:"language"`   // "go", or "" when not analyzed for complexity
	Commits    int       `json:"commits"`    // churn: commits touching the file
	LastAuthor string    `json:"lastAuthor,omitempty"`
	LastChange time.Time `json:"lastChange,omitempty"`
	Functions  int       `json:"functions"`  // functions parsed
	Cyclomatic int       `json:"cyclomatic"` // summed over functions
	Cognitive  int       `json:"cognitive"`  // summed over functions

	// Complexity is the magnitude fed to the scorer: total cognitive when any
	// function reported it, else total cyclomatic.
	Complexity float64 `json:"complexity"`

	// Normalized 0..1 across the analyzed set, and their product.
	ChurnNorm      float64 `json:"churnNorm"`
	ComplexityNorm float64 `json:"complexityNorm"`
	Score          float64 `json:"score"`
}

// Report is the outcome of an [Analyze] run: Files is sorted by Score
// descending (ties broken by Complexity, then Commits, then Path).
type Report struct {
	Root      string     `json:"root"`
	HeadSHA   string     `json:"headSha,omitempty"`
	Generated time.Time  `json:"generated"`
	Files     []FileRisk `json:"files"`
}

// Options tunes an [Analyze] run.
type Options struct {
	// IncludeUntracked scores files not tracked by git (they have zero churn,
	// so they rarely rank — off by default keeps the report to real code).
	IncludeUntracked bool
	// now is injected in tests; nil uses time.Now.
	now func() time.Time
}

// Analyze walks root, collects churn and complexity for each candidate source
// file, and returns a scored, ranked [Report]. root need not be a git
// repository — without git, churn is zero and the report degrades to a pure
// complexity ranking.
func Analyze(ctx context.Context, root string, opts Options) (Report, error) {
	churn, headSHA, err := newGitmetaChurn(ctx, root)
	if err != nil {
		return Report{}, err
	}
	files, err := collect(root, churn, goComplexity{}, opts)
	if err != nil {
		return Report{}, err
	}
	Score(files)

	nowFn := opts.now
	if nowFn == nil {
		nowFn = time.Now
	}
	return Report{
		Root:      root,
		HeadSHA:   headSHA,
		Generated: nowFn(),
		Files:     files,
	}, nil
}
