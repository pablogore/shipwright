package golang

import (
	"context"
	"encoding/json"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestGoRuntimeUpgrader_Upgrade_NilClient(t *testing.T) {
	upgrader := &GoRuntimeUpgrader{}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dagger client is not configured")
	assert.Nil(t, out)
}

func TestGoRuntimeUpgrader_Upgrade_NilSource(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), nil, "1.27.0")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "source directory is nil")
	assert.Nil(t, out)
}

// TestGoRuntimeUpgrader_Upgrade_SingleModule_HappyPath proves the
// discovery-driven single-module mutation flow: go.mod's go directive is
// updated, no .go-version WithNewFile call happens because the fixture has
// none (missing-location-skipped — never fabricated), and the mutated
// Directory's real handle is returned via the mocked WithNewFile chain.
func TestGoRuntimeUpgrader_Upgrade_SingleModule_HappyPath(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}
	afterMod.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Return(afterMod)
	afterMod.On("GetRealDirectory").Return(realDir)

	var capturedModContents string
	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedModContents = args.String(1)
	}).Return(afterMod)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)
	assert.Contains(t, capturedModContents, "go 1.27.0")

	mockDir.AssertExpectations(t)
	afterMod.AssertExpectations(t)
	// .go-version was never present in the single-module fixture, so it
	// must never be written — asserting it was NOT called proves
	// discovery-driven mutation, not a fabricated write.
	afterMod.AssertNotCalled(t, "WithNewFile", ".go-version", mock.Anything)
}

// TestGoRuntimeUpgrader_Upgrade_Downgrade_AmbiguousAbort proves the
// no-partial-mutation guarantee: a rejected downgrade (A4) returns (nil,
// err) before any WithNewFile call, never a partially mutated Directory.
func TestGoRuntimeUpgrader_Upgrade_Downgrade_AmbiguousAbort(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "downgrade")
	withMockDirectory(t, mockDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.20.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var ambiguous *AmbiguousToolchainError
	require.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, CodeA4, ambiguous.Code)

	mockDir.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}

// TestGoRuntimeUpgrader_Upgrade_Malformed_AmbiguousAbort proves the same
// no-partial-mutation guarantee for A5 (malformed existing go.mod).
func TestGoRuntimeUpgrader_Upgrade_Malformed_AmbiguousAbort(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "malformed")
	withMockDirectory(t, mockDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var ambiguous *AmbiguousToolchainError
	require.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, CodeA5, ambiguous.Code)

	mockDir.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}

// TestUpgrade_PathEscape is the RED test tasks.md 3.2 requires: a go.work
// use directive that resolves outside the workspace root (../../etc) must
// abort before any WithNewFile call — proven via a mock assertion, not
// just the returned error (threat matrix: path traversal via go.work
// use). Placed here rather than toolchain_test.go (tasks.md's literal
// file assignment) because a mock-based "no WithNewFile call happened"
// assertion needs daggerkit's mocked DaggerDirectory, which toolchain.go's
// pure-Go test file deliberately never imports (D-9's read/write seam
// stays mock-testable only in this package's Dagger-facing test files);
// see apply-progress deviation notes.
func TestUpgrade_PathEscape(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "path-escape")
	withMockDirectory(t, mockDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var ambiguous *AmbiguousToolchainError
	require.ErrorAs(t, err, &ambiguous)
	assert.Equal(t, CodeA7, ambiguous.Code)

	mockDir.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}

// TestUpgrade_Workspace proves the multi-module go.work mutation loop
// (tasks.md 3.4/3.5): every use'd module's go.mod, plus go.work's own go
// directive, are mutated, and the report names every module's outcome.
// This is also the test tasks.md 3.6's regression-guard verification step
// names explicitly (`go test -run TestUpgrade_Workspace`).
func TestUpgrade_Workspace(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "workspace-3-modules")
	withMockDirectory(t, mockDir)

	afterWork := &daggerkit.MockDaggerDirectory{}
	afterVersion := &daggerkit.MockDaggerDirectory{}
	afterA := &daggerkit.MockDaggerDirectory{}
	afterB := &daggerkit.MockDaggerDirectory{}
	afterC := &daggerkit.MockDaggerDirectory{}
	afterReport := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	var capturedWork, capturedVersion, capturedA, capturedB, capturedC, capturedReport string

	mockDir.On("WithNewFile", "go.work", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedWork = args.String(1)
	}).Return(afterWork)
	// The fixture also has a root .go-version, mutated the same
	// discovery-driven way the single-module path does.
	afterWork.On("WithNewFile", ".go-version", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedVersion = args.String(1)
	}).Return(afterVersion)
	afterVersion.On("WithNewFile", "modA/go.mod", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedA = args.String(1)
	}).Return(afterA)
	afterA.On("WithNewFile", "modB/go.mod", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedB = args.String(1)
	}).Return(afterB)
	afterB.On("WithNewFile", "modC/go.mod", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedC = args.String(1)
	}).Return(afterC)
	afterC.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(afterReport)
	afterReport.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)

	assert.Contains(t, capturedWork, "go 1.27.0")
	assert.Equal(t, "1.27.0\n", capturedVersion)
	assert.Contains(t, capturedA, "go 1.27.0")
	assert.Contains(t, capturedB, "go 1.27.0")
	assert.Contains(t, capturedC, "go 1.27.0")

	var report shipwright.UpgradeReport
	require.NoError(t, json.Unmarshal([]byte(capturedReport), &report))
	assert.Equal(t, "1.27.0", report.TargetVersion)
	require.Len(t, report.Modules, 3)

	byPath := make(map[string]shipwright.ModuleDrift, len(report.Modules))
	for _, m := range report.Modules {
		byPath[m.Path] = m
	}
	for _, p := range []string{"modA", "modB", "modC"} {
		m, ok := byPath[p]
		require.Truef(t, ok, "report missing module %s", p)
		assert.Equal(t, "1.26.7", m.PreviousGo)
		assert.Equal(t, "1.27.0", m.UpdatedGo)
	}

	mockDir.AssertExpectations(t)
	afterWork.AssertExpectations(t)
	afterVersion.AssertExpectations(t)
	afterA.AssertExpectations(t)
	afterB.AssertExpectations(t)
	afterC.AssertExpectations(t)
	afterReport.AssertExpectations(t)
}

// TestGoRuntimeUpgrader_Upgrade_WritesUpgradeReport proves the report
// content written to runtimeUpgradeReportPath matches
// shipwright.UpgradeReport's contract.
func TestGoRuntimeUpgrader_Upgrade_WritesUpgradeReport(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	var capturedReport string
	afterMod.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)

	var report shipwright.UpgradeReport
	require.NoError(t, json.Unmarshal([]byte(capturedReport), &report))
	assert.Equal(t, ".", report.WorkspaceRoot)
	assert.Equal(t, "1.27.0", report.TargetVersion)
	require.Len(t, report.Modules, 1)
	assert.Equal(t, ".", report.Modules[0].Path)
	assert.Equal(t, "1.26.7", report.Modules[0].PreviousGo)
	assert.Equal(t, "1.27.0", report.Modules[0].UpdatedGo)
}
