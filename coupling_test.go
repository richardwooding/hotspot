package hotspot

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Proves coupling is genuinely multi-language: a Python project (not Go) whose
// import graph go-coupling resolves via its Python adapter, with imports
// extracted by treesitter-symbols. web imports core, so web has efferent
// coupling and core has afferent coupling.
func TestCouplingIsMultiLanguage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = \"app\"\n")
	writeFile(t, filepath.Join(root, "app", "web", "v.py"), "import app.core\n\ndef view():\n    return app.core.load()\n")
	writeFile(t, filepath.Join(root, "app", "core", "c.py"), "import os\n\ndef load():\n    return os.getcwd()\n")

	files := []FileRisk{
		{Path: "app/web/v.py", Language: "python"},
		{Path: "app/core/c.py", Language: "python"},
	}
	per := goCoupling{}.Analyze(root, files)
	if len(per) == 0 {
		t.Fatal("no coupling resolved for a Python project")
	}

	web, wok := per["app/web/v.py"]
	core, cok := per["app/core/c.py"]
	if !wok || !cok {
		t.Fatalf("missing files in coupling map: web=%v core=%v (got %v)", wok, cok, per)
	}
	if web.Efferent < 1 {
		t.Errorf("web should import core (Ce>=1), got %+v", web)
	}
	if core.Afferent < 1 {
		t.Errorf("core should be imported by web (Ca>=1), got %+v", core)
	}
}
