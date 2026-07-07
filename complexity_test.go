package hotspot

import "testing"

// Proves the complexity provider is genuinely multi-language: Go via go/ast and
// a non-Go language (Python) via codemetrics' pure-Go tree-sitter backend.
func TestComplexityIsMultiLanguage(t *testing.T) {
	c := codemetricsComplexity{}

	if lang, ok := c.Language("x/foo.go"); !ok || lang != "go" {
		t.Fatalf("expected go for .go, got %q ok=%v", lang, ok)
	}
	lang, ok := c.Language("x/foo.py")
	if !ok || lang != "python" {
		t.Fatalf("expected python for .py, got %q ok=%v", lang, ok)
	}

	// A Python function with a branch: cyclomatic should exceed 1.
	fns, cyc, _, cok := c.Complexity("python", []byte("def f(x):\n    if x:\n        return 1\n    return 0\n"))
	if !cok || fns < 1 || cyc < 2 {
		t.Fatalf("python complexity via tree-sitter failed: fns=%d cyc=%d ok=%v", fns, cyc, cok)
	}

	if len(SupportedLanguages()) < 10 {
		t.Fatalf("expected many supported languages, got %d: %v", len(SupportedLanguages()), SupportedLanguages())
	}
}
