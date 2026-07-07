package hotspot

import (
	"os"
	"path/filepath"

	coupling "github.com/richardwooding/go-coupling"
	symbols "github.com/richardwooding/treesitter-symbols"
)

// Coupling is one node's afferent/efferent profile, as produced by go-coupling
// and attached to every file in that node.
type Coupling struct {
	Afferent    int     `json:"afferent"`    // Ca: first-party nodes importing this one
	Efferent    int     `json:"efferent"`    // Ce: first-party nodes this one imports
	Instability float64 `json:"instability"` // I = Ce/(Ca+Ce)
}

// CouplingProvider computes per-file coupling for a whole project, keyed by the
// same repo-relative slash path as FileRisk.Path. A nil/empty result means the
// project's ecosystem is not analysable (e.g. no recognised build manifest), in
// which case the score falls back to churn × complexity.
type CouplingProvider interface {
	Analyze(root string, files []FileRisk) map[string]Coupling
}

// goCoupling computes package coupling across every language go-coupling
// supports. It extracts each file's imports/package with treesitter-symbols
// (pure Go, 17 languages), hands them to go-coupling — whose adapter for the
// project's ecosystem (Go/Rust/Python/JS-TS/Ruby/C-C++/Swift, or the
// declaration-based Java/Kotlin/Scala/C#/PHP/Perl) turns them into a first-party
// import graph — then joins the per-node Ca/Ce back to files via FileCoupling.
type goCoupling struct{}

func (goCoupling) Analyze(root string, files []FileRisk) map[string]Coupling {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}

	cfiles := make([]coupling.File, 0, len(files))
	abs := make(map[string]string, len(files)) // FileRisk.Path -> absolute path (the go-coupling key)
	for _, f := range files {
		p := filepath.Join(absRoot, filepath.FromSlash(f.Path))
		src, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		sym, extractErr := symbols.Extract(f.Language, src)
		if extractErr != nil {
			continue // language treesitter-symbols doesn't handle: no coupling, still scored on churn × complexity
		}
		abs[f.Path] = p
		cfiles = append(cfiles, coupling.File{
			Path:            p,
			Language:        f.Language,
			Imports:         sym.Imports,
			RelativeImports: sym.RelativeImports,
			Package:         sym.Package,
		})
	}

	g := coupling.Build(absRoot, cfiles)
	if !g.Analysable() {
		return nil
	}
	byAbs := g.FileCoupling()

	out := make(map[string]Coupling, len(byAbs))
	for _, f := range files {
		if c, ok := byAbs[abs[f.Path]]; ok {
			out[f.Path] = Coupling{Afferent: c.Afferent, Efferent: c.Efferent, Instability: c.Instability}
		}
	}
	return out
}

// attachCoupling fills each file's Afferent/Efferent/Instability from its node's
// coupling profile. It is a no-op when the project's ecosystem is not analysable,
// so the score simply falls back to churn × complexity.
func attachCoupling(files []FileRisk, cp CouplingProvider, root string) {
	per := cp.Analyze(root, files)
	for i := range files {
		if c, ok := per[files[i].Path]; ok {
			files[i].Afferent = c.Afferent
			files[i].Efferent = c.Efferent
			files[i].Instability = c.Instability
		}
	}
}
