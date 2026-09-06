// TestExamples_ParseAndBuildGraph is the drift guard for docs/WORKFLOW_GUIDE.md
// (design.md "Drift guard", tasks.md doc-003 1.2): every manifest under
// examples/workflow/ must actually parse (stages 1-3) and compile into a
// graph (stage 5), so the example files the guide links to and inlines are
// executable specifications, not prose that can silently drift from the
// schema.
package manifest_test

import (
	"path/filepath"
	"testing"

	"github.com/pablogore/shipwright/internal/workflow/graph"
	"github.com/pablogore/shipwright/internal/workflow/manifest"
)

func TestExamples_ParseAndBuildGraph(t *testing.T) {
	matches, err := filepath.Glob("../../../examples/workflow/*.yaml")
	if err != nil {
		t.Fatalf("Glob() error = %v, want nil", err)
	}
	if len(matches) == 0 {
		t.Fatal("Glob() matched no example manifests, want at least one")
	}

	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			m, err := manifest.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile(%q) error = %v, want nil", path, err)
			}

			if _, err := graph.Build(m.Spec.Steps); err != nil {
				t.Fatalf("graph.Build(%q) error = %v, want nil", path, err)
			}
		})
	}
}
