package hotspot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeChurn reports commits for a fixed set of repo-relative paths.
type fakeChurn struct {
	root    string
	commits map[string]int // rel path -> commits (present = tracked)
}

func (f fakeChurn) Churn(abs string) (int, string, time.Time, bool) {
	rel, _ := filepath.Rel(f.root, abs)
	rel = filepath.ToSlash(rel)
	c, ok := f.commits[rel]
	return c, "", time.Time{}, ok
}

// fakeCmp treats .go files as language "go" with complexity = file length.
type fakeCmp struct{}

func (fakeCmp) Language(path string) (string, bool) {
	if filepath.Ext(path) == ".go" {
		return "go", true
	}
	return "", false
}
func (fakeCmp) Complexity(_ string, src []byte) (int, int, int, bool) {
	return 1, len(src), len(src), true
}

func TestCollectWalksAndFilters(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.go", "package a")
	write("pkg/b.go", "package b\n// more")
	write("README.md", "# not source")       // wrong language -> skipped
	write("vendor/dep.go", "package dep")     // skipped dir
	write(".git/config", "x")                 // skipped dir
	write("untracked.go", "package u")        // no churn record

	churn := fakeChurn{root: root, commits: map[string]int{
		"a.go": 10, "pkg/b.go": 3,
	}}

	got, err := collect(root, churn, fakeCmp{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tracked .go files, got %d: %v", len(got), order(got))
	}
	// With IncludeUntracked the untracked .go file joins in.
	got2, err := collect(root, churn, fakeCmp{}, Options{IncludeUntracked: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got2) != 3 {
		t.Fatalf("expected 3 files with untracked, got %d: %v", len(got2), order(got2))
	}
}
