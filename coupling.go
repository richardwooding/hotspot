package hotspot

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	coupling "github.com/richardwooding/go-coupling"
)

// Coupling is one package's afferent/efferent profile, as produced by
// go-coupling and attached to every file in that package.
type Coupling struct {
	Afferent    int     `json:"afferent"`    // Ca: first-party packages importing this one
	Efferent    int     `json:"efferent"`    // Ce: first-party packages this one imports
	Instability float64 `json:"instability"` // I = Ce/(Ca+Ce)
}

// CouplingProvider computes package coupling for a whole project and maps a
// repo-relative file path to its package node. ok is false when the project's
// ecosystem is not analysable (e.g. no go.mod).
type CouplingProvider interface {
	Analyze(root string) (perNode map[string]Coupling, nodeOf func(relPath string) (string, bool), ok bool)
}

// goCoupling builds a Go import graph with go-coupling. It extracts each file's
// imports with go/parser (imports only — cheap), then joins results back to
// files using the same node rule go-coupling uses: module + the file's
// directory relative to root.
type goCoupling struct{}

func (goCoupling) Analyze(root string) (map[string]Coupling, func(string) (string, bool), bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, false
	}
	module := goModulePath(absRoot)
	if module == "" {
		return nil, nil, false // not a Go module: nothing to resolve first-party against
	}

	var files []coupling.File
	fset := token.NewFileSet()
	_ = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != absRoot && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(p)) != ".go" {
			return nil
		}
		src, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		af, parseErr := parser.ParseFile(fset, p, src, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		imps := make([]string, 0, len(af.Imports))
		for _, spec := range af.Imports {
			if s, uErr := strconv.Unquote(spec.Path.Value); uErr == nil {
				imps = append(imps, s)
			}
		}
		files = append(files, coupling.File{Path: p, Language: "go", Imports: imps})
		return nil
	})

	perNode := make(map[string]Coupling, 16)
	for _, c := range coupling.Analyze(absRoot, files) {
		perNode[c.Package] = Coupling{Afferent: c.Afferent, Efferent: c.Efferent, Instability: c.Instability}
	}

	nodeOf := func(relPath string) (string, bool) {
		dir := path.Dir(filepath.ToSlash(relPath))
		if dir == "." || dir == "" {
			return module, true
		}
		return module + "/" + dir, true
	}
	return perNode, nodeOf, true
}

// attachCoupling fills each file's Afferent/Efferent/Instability from its
// package's coupling profile. It is a no-op when the project's ecosystem is
// not analysable, so the score simply falls back to churn × complexity.
func attachCoupling(files []FileRisk, cp CouplingProvider, root string) {
	perNode, nodeOf, ok := cp.Analyze(root)
	if !ok {
		return
	}
	for i := range files {
		node, ok := nodeOf(files[i].Path)
		if !ok {
			continue
		}
		if c, ok := perNode[node]; ok {
			files[i].Afferent = c.Afferent
			files[i].Efferent = c.Efferent
			files[i].Instability = c.Instability
		}
	}
}

// goModulePath reads the module path from root/go.mod, or "" if absent.
func goModulePath(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}
