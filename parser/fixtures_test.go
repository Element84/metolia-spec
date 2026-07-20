package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDecodeFixtureCorpus decodes every schema fixture in the repository.
// All fixtures, valid and invalid alike, are well-formed JSON — the invalid
// ones violate structure, not syntax — so the decode layer must accept every
// one without diagnostics.
func TestDecodeFixtureCorpus(t *testing.T) {
	paths, err := filepath.Glob("../tests/schemas/*/*/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixtures found; expected the repository corpus under ../tests/schemas/")
	}
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		v, diags := decode(src, filepath.Base(path))
		if len(diags) != 0 {
			t.Errorf("%s: unexpected diagnostics: %v", path, diags)
		}
		if v == nil {
			t.Errorf("%s: nil tree", path)
		}
	}
}
