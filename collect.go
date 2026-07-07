package hotspot

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/richardwooding/codemetrics"
	"github.com/richardwooding/gitmeta"
)

// ChurnProvider yields how often a file has changed. absPath is an absolute
// path; ok is false when the provider has no record (e.g. untracked).
type ChurnProvider interface {
	Churn(absPath string) (commits int, author string, at time.Time, ok bool)
}

// ComplexityProvider computes complexity for one file's contents. lang is the
// detected language identifier; ok is false when the language is unsupported.
type ComplexityProvider interface {
	Language(path string) (lang string, supported bool)
	Complexity(lang string, src []byte) (functions, cyclomatic, cognitive int, ok bool)
}

// skipDirs are never descended into.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".idea": true, ".vscode": true, "dist": true, "build": true,
}

// collect walks root and builds one (unscored) FileRisk per candidate source
// file. A file is a candidate when the complexity provider recognizes its
// language; files with no churn record are kept only when opts.IncludeUntracked.
func collect(root string, churn ChurnProvider, cmp ComplexityProvider, opts Options) ([]FileRisk, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var out []FileRisk
	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != absRoot && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		lang, ok := cmp.Language(path)
		if !ok {
			return nil
		}
		commits, author, at, tracked := churn.Churn(path)
		if !tracked && !opts.IncludeUntracked {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // unreadable file: skip, don't fail the whole walk
		}
		fns, cyc, cog, cok := cmp.Complexity(lang, src)
		if !cok {
			return nil
		}
		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			rel = path
		}
		fr := FileRisk{
			Path:       filepath.ToSlash(rel),
			Language:   lang,
			Commits:    commits,
			LastAuthor: author,
			LastChange: at,
			Functions:  fns,
			Cyclomatic: cyc,
			Cognitive:  cog,
		}
		// Cognitive better reflects human effort; fall back to cyclomatic when a
		// language reports no cognitive score.
		if cog > 0 {
			fr.Complexity = float64(cog)
		} else {
			fr.Complexity = float64(cyc)
		}
		out = append(out, fr)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

// --- gitmeta adapter (churn) -------------------------------------------------

type gitmetaChurn struct{ cache *gitmeta.Cache }

// newGitmetaChurn scans root once. When root is not a git working tree it
// returns a no-op provider (every file "untracked") and an empty HEAD SHA,
// so Analyze still runs as a pure complexity ranking.
func newGitmetaChurn(ctx context.Context, root string) (ChurnProvider, string, error) {
	cache, err := gitmeta.New(ctx, root)
	if err != nil {
		return nil, "", err
	}
	if cache == nil {
		return gitmetaChurn{}, "", nil
	}
	return gitmetaChurn{cache: cache}, cache.HeadSHA(), nil
}

func (g gitmetaChurn) Churn(absPath string) (int, string, time.Time, bool) {
	if g.cache == nil {
		return 0, "", time.Time{}, false
	}
	info, ok := g.cache.Lookup(absPath)
	if !ok {
		return 0, "", time.Time{}, false
	}
	return info.CommitCount, info.LastCommitAuthor, info.LastCommitTime, true
}

// --- codemetrics adapter (complexity) ----------------------------------------

// goComplexity maps source files to codemetrics. codemetrics' library API
// currently parses Go (via go/ast); other languages return unsupported until
// codemetrics widens its library surface, at which point extByLang grows.
type goComplexity struct{}

var extByLang = map[string]string{
	".go": "go",
}

func (goComplexity) Language(path string) (string, bool) {
	lang, ok := extByLang[strings.ToLower(filepath.Ext(path))]
	return lang, ok
}

func (goComplexity) Complexity(lang string, src []byte) (int, int, int, bool) {
	fns, err := codemetrics.Parse(lang, src)
	if err != nil {
		return 0, 0, 0, false
	}
	var cyc, cog int
	for _, f := range fns {
		cyc += f.Cyclomatic
		if f.Cognitive != nil {
			cog += *f.Cognitive
		}
	}
	return len(fns), cyc, cog, true
}
