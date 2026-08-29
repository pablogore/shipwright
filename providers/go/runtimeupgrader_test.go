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

// TestGoRuntimeUpgrader_Upgrade_GoWork_NotYetSupported proves Phase 2's
// explicit scope boundary: a workspace with a go.work at its root is
// rejected outright (fail-closed), never partially traversed.
func TestGoRuntimeUpgrader_Upgrade_GoWork_NotYetSupported(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockDir := mockDirFromFixture(t, "workspace-3-modules")
	withMockDirectory(t, mockDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "not yet supported")

	mockDir.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
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
