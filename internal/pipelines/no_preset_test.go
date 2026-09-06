package pipelines

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot resolves the repository root from this test file's own location
// (internal/pipelines/no_preset_test.go is two directories below root),
// rather than relying on the test binary's working directory, which `go
// test` sets per-package.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller for repoRoot")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// TestNoPresetRegistryOrStackNamedPipelineSurface is task 11.1's RED test
// (design.md D-F, D-N; tasks.md 11.1): after this work unit, no preset
// registry, factory map, or capability-set implementation keyed by a stack
// name (e.g. "go-service") may exist anywhere in the tree. It fails while
// the legacy preset registry/contract/implementations still exist, and
// passes once they are deleted (tasks.md 11.3).
//
// Scope, judgment call recorded for sdd-verify: internal/pipelines/infra/**
// is included here even though tasks.md/design.md name only
// internal/pipelines/{pipeline,registry,options}.go, common/interfaces.go,
// and go-service/** explicitly. infra/** is a second stack-named preset
// pipeline (SyntegrityInfraPipeline) that depends on exactly the
// pipelines.Pipeline/pipelines.HookFunc contract this test also asserts is
// gone; it was only ever reachable through the same PipelineRegistry this
// work unit deletes, so it is now equally dead and equally a violation of
// this invariant.
func TestNoPresetRegistryOrStackNamedPipelineSurface(t *testing.T) {
	root := repoRoot(t)

	forbiddenPaths := []string{
		filepath.Join("internal", "pipelines", "pipeline.go"),
		filepath.Join("internal", "pipelines", "registry.go"),
		filepath.Join("internal", "pipelines", "common"),
		filepath.Join("internal", "pipelines", "go-service"),
		filepath.Join("internal", "pipelines", "infra"),
	}

	for _, rel := range forbiddenPaths {
		abs := filepath.Join(root, rel)
		if _, err := os.Stat(abs); !os.IsNotExist(err) {
			t.Errorf("forbidden legacy preset surface still exists: %s (design.md D-F/D-N, tasks.md 11.1/11.3)", rel)
		}
	}
}
