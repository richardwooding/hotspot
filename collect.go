package hotspot

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/richardwooding/codemetrics"
	"github.com/richardwooding/codemetrics/treesitter"
	"github.com/richardwooding/gitmeta"
	"github.com/richardwooding/projectdetect"
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

// codemetricsComplexity computes per-function complexity for every language
// codemetrics supports: Go via go/ast, and the rest via codemetrics' pure-Go
// tree-sitter backend (github.com/odvcencio/gotreesitter — no cgo). It uses the
// exact pipeline the codemetrics CLI uses: projectdetect resolves the language
// from the path, then Go dispatches to ParseGo and everything else to
// treesitter.Parse.
type codemetricsComplexity struct{}

// supportedLangs is the set hotspot can score: "go" plus codemetrics' tree-sitter
// languages. Built once.
var supportedLangs = func() map[string]bool {
	m := map[string]bool{"go": true}
	for _, l := range treesitter.SupportedLanguages() {
		m[l] = true
	}
	return m
}()

// SupportedLanguages returns the sorted language identifiers hotspot scores for
// complexity (Go + codemetrics' tree-sitter set).
func SupportedLanguages() []string {
	out := make([]string, 0, len(supportedLangs))
	for l := range supportedLangs {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func (codemetricsComplexity) Language(path string) (string, bool) {
	lang := projectdetect.LanguageForPath(path)
	if lang == "" || !supportedLangs[lang] {
		return "", false
	}
	return lang, true
}

func (codemetricsComplexity) Complexity(lang string, src []byte) (int, int, int, bool) {
	var (
		fns []codemetrics.FunctionMetrics
		err error
	)
	if lang == "go" {
		fns, err = codemetrics.ParseGo(src)
	} else {
		fns, err = treesitter.Parse(lang, src)
	}
	if err != nil { // unsupported/unavailable language or hard parse failure
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
