package golang

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// mockDirFromFixture builds a daggerkit.MockDaggerDirectory backed by a
// testdata/runtime/<name> fixture: Entries returns the fixture root's
// immediate child names (mirroring real dagger.Directory.Entries), and
// File(path) is wired for every file found anywhere under the fixture root,
// keyed by its slash-separated path relative to the fixture root. This is
// the mock-level counterpart of toolchain_test.go's loadFixture (double-
// selection order rule 1: daggerkit mocks first, no live engine needed).
func mockDirFromFixture(t *testing.T, name string) *daggerkit.MockDaggerDirectory {
	t.Helper()

	root := filepath.Join("testdata", "runtime", name)

	rootEntries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("failed to read fixture root %s: %v", name, err)
	}
	names := make([]string, 0, len(rootEntries))
	for _, e := range rootEntries {
		names = append(names, e.Name())
	}

	mockDir := &daggerkit.MockDaggerDirectory{}
	mockDir.On("Entries", mock.Anything).Return(names, nil)

	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		mockFile := &daggerkit.MockDaggerFile{}
		mockFile.On("Contents", mock.Anything).Return(string(data), nil)
		mockDir.On("File", rel).Return(mockFile)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("failed to walk fixture %s: %v", name, walkErr)
	}

	return mockDir
}

// withMockDirectory overrides the package-level newDaggerDirectory seam so
// Inspect reads mockDir instead of trying to talk to a live Dagger engine,
// restoring the original value once the test completes.
func withMockDirectory(t *testing.T, mockDir *daggerkit.MockDaggerDirectory) {
	t.Helper()
	original := newDaggerDirectory
	newDaggerDirectory = func(*dagger.Directory) daggerkit.DaggerDirectory { return mockDir }
	t.Cleanup(func() { newDaggerDirectory = original })
}

func TestGoRuntimeInspector_Inspect_NilClient(t *testing.T) {
	inspector := &GoRuntimeInspector{}

	out, err := inspector.Inspect(context.Background(), &dagger.Directory{})

	if err == nil {
		t.Fatal("Inspect() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("Inspect() error = %v, want it to mention an unconfigured client", err)
	}
	if out != "" {
		t.Fatalf("Inspect() report = %q, want empty on error", out)
	}
}

func TestGoRuntimeInspector_Inspect_NilSource(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	inspector := &GoRuntimeInspector{Client: mockClient}

	out, err := inspector.Inspect(context.Background(), nil)

	if err == nil {
		t.Fatal("Inspect() error = nil, want error for nil source")
	}
	if !strings.Contains(err.Error(), "source directory is nil") {
		t.Fatalf("Inspect() error = %v, want it to mention a nil source", err)
	}
	if out != "" {
		t.Fatalf("Inspect() report = %q, want empty on error", out)
	}
}

// TestGoRuntimeInspector_Inspect_SingleModule proves the happy path over a
// single-module workspace (no go.work), and the spec's "Missing declarative
// location is omitted, not fabricated" scenario: no go.work in the fixture
// means no "go.work" entry in the report's Sources map.
func TestGoRuntimeInspector_Inspect_SingleModule(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	inspector := &GoRuntimeInspector{Client: mockClient}

	out, err := inspector.Inspect(context.Background(), &dagger.Directory{})
	require.NoError(t, err)

	var report shipwright.DriftReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))

	assert.Equal(t, ".", report.WorkspaceRoot)
	assert.Empty(t, report.Sources, "no go.work/.go-version in the fixture, so Sources must be empty, not fabricated")
	assert.False(t, report.Conflict.Ambiguous)
	require.Len(t, report.Modules, 1)
	assert.Equal(t, ".", report.Modules[0].Path)
	assert.Equal(t, "1.26.7", report.Modules[0].Go)

	mockDir.AssertExpectations(t)
}

// TestGoRuntimeInspector_Inspect_Workspace proves the multi-module,
// unanimous-versions happy path (go.work + .go-version present and
// agreeing with every module), and that ExpectedVersion is echoed as
// configured.
func TestGoRuntimeInspector_Inspect_Workspace(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "workspace-3-modules")
	withMockDirectory(t, mockDir)

	inspector := &GoRuntimeInspector{Client: mockClient, ExpectedVersion: "1.26.7"}

	out, err := inspector.Inspect(context.Background(), &dagger.Directory{})
	require.NoError(t, err)

	var report shipwright.DriftReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))

	assert.Equal(t, "1.26.7", report.ExpectedVersion)
	assert.Equal(t, "1.26.7", report.Sources["go.work"])
	assert.Equal(t, "1.26.7", report.Sources[".go-version"])
	assert.False(t, report.Conflict.Ambiguous)
	assert.Len(t, report.Modules, 3)

	mockDir.AssertExpectations(t)
}

// TestGoRuntimeInspector_Inspect_Ambiguous_ReportsWithoutFailing proves the
// spec's "Ambiguous sources are reported, never guessed" scenario: with
// failOnDrift left at its false default, Inspect still succeeds and the
// conflict is explicit in the report, naming both sources.
func TestGoRuntimeInspector_Inspect_Ambiguous_ReportsWithoutFailing(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "divergent-go")
	withMockDirectory(t, mockDir)

	inspector := &GoRuntimeInspector{Client: mockClient}

	out, err := inspector.Inspect(context.Background(), &dagger.Directory{})
	require.NoError(t, err)

	var report shipwright.DriftReport
	require.NoError(t, json.Unmarshal([]byte(out), &report))

	assert.True(t, report.Conflict.Ambiguous)
	assert.Equal(t, CodeA1, report.Conflict.Code)
	assert.NotEmpty(t, report.Conflict.Sites)
}

// TestGoRuntimeInspector_Inspect_Ambiguous_FailsWhenConfigured proves
// design.md D-4b: failOnDrift: true turns the same conflict into a
// returned error instead of a report-only success.
func TestGoRuntimeInspector_Inspect_Ambiguous_FailsWhenConfigured(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "divergent-go")
	withMockDirectory(t, mockDir)

	inspector := &GoRuntimeInspector{Client: mockClient, FailOnDrift: true}

	out, err := inspector.Inspect(context.Background(), &dagger.Directory{})
	require.Error(t, err)
	assert.Empty(t, out)

	var ambiguous *AmbiguousToolchainError
	require.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, CodeA1, ambiguous.Code)
}
