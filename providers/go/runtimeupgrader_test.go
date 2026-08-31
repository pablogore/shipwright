package golang

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// validatedGoMod is a minimal, syntactically valid go.mod used as the
// "post go mod tidy" content the mocked validation container returns via
// File(".../go.mod").Contents (design.md D-7's recordModuleDelta reads it
// to compute the report's per-module require-list delta). It carries one
// require directive absent from every testdata/runtime fixture's own
// go.mod, so tests can assert AddedModules is populated without needing a
// real `go mod tidy` run.
const validatedGoMod = "module example.com/fixture/validated\n\ngo 1.27.0\n\nrequire example.com/dep v1.0.0\n"

// unchangedGoSum and changedGoSum are the "before" and post-tidy go.sum
// contents used to drive GoSumChanged via a real byte comparison
// (recordModuleDelta), rather than the go.mod require-list delta a
// GoSumChanged consumer must never rely on.
const unchangedGoSum = "example.com/dep v1.0.0 h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\nexample.com/dep v1.0.0/go.mod h1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb=\n"
const changedGoSum = "example.com/dep v1.0.1 h1:ccccccccccccccccccccccccccccccccccccccccccc=\nexample.com/dep v1.0.1/go.mod h1:ddddddddddddddddddddddddddddddddddddddddddd=\n"

// mockFileContents returns a MockDaggerFile whose Contents(ctx) resolves
// to contents.
func mockFileContents(contents string) *daggerkit.MockDaggerFile {
	f := &daggerkit.MockDaggerFile{}
	f.On("Contents", mock.Anything).Return(contents, nil)
	return f
}

// mockFileNotFound returns a MockDaggerFile whose Contents(ctx) fails the
// same way a missing path's read surfaces from a real Dagger engine
// ("no such file or directory") — recordModuleDelta's go.sum comparison
// treats that as a legitimate absent state, not a hard failure.
func mockFileNotFound() *daggerkit.MockDaggerFile {
	f := &daggerkit.MockDaggerFile{}
	f.On("Contents", mock.Anything).Return("", errors.New("open go.sum: no such file or directory"))
	return f
}

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
// none (missing-location-skipped — never fabricated), the mutated
// Directory is mounted into a "golang:"+targetVersion validation container
// (design.md D-7), and the container's exported workdir's real handle is
// returned via the mocked Directory/WithNewFile chain.
func TestGoRuntimeUpgrader_Upgrade_SingleModule_HappyPath(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	var capturedModContents string
	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedModContents = args.String(1)
	}).Return(afterMod)

	afterModFile := &daggerkit.MockDaggerFile{}
	afterModFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	mockDir.On("File", "go.sum").Return(mockFileContents(unchangedGoSum))

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileContents(unchangedGoSum))
	mockContainer.On("Directory", "/workspace").Return(finalDir)
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)
	assert.Contains(t, capturedModContents, "go 1.27.0")

	mockDir.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
	// .go-version was never present in the single-module fixture, so it
	// must never be written — asserting it was NOT called proves
	// discovery-driven mutation, not a fabricated write.
	afterMod.AssertNotCalled(t, "WithNewFile", ".go-version", mock.Anything)
	// No "go mod tidy" exec: Tidy defaults to false on the struct itself
	// (design.md's manifest-level default(true) is applied at
	// registration binding, not here — see register.go).
	mockContainer.AssertNotCalled(t, "WithExec", []string{"go", "mod", "tidy"}, mock.Anything)
}

// TestGoRuntimeUpgrader_Upgrade_TidyEnabled_RunsGoModTidyBeforeBuild proves
// design.md D-7's ordering: when Tidy is true, `go mod tidy` runs before
// `go build ./...` for the module (an untidied go.sum would otherwise
// break the build).
func TestGoRuntimeUpgrader_Upgrade_TidyEnabled_RunsGoModTidyBeforeBuild(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	afterModFile := &daggerkit.MockDaggerFile{}
	afterModFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	mockDir.On("File", "go.sum").Return(mockFileContents(unchangedGoSum))

	var order []string
	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "mod", "tidy"}, daggerkit.DaggerContainerWithExecOpts{}).Run(func(mock.Arguments) {
		order = append(order, "tidy")
	}).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Run(func(mock.Arguments) {
		order = append(order, "build")
	}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileContents(unchangedGoSum))
	mockContainer.On("Directory", "/workspace").Return(finalDir)
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient, Tidy: true}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)
	assert.Equal(t, []string{"tidy", "build"}, order)

	mockContainer.AssertExpectations(t)
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
// (tasks.md 3.4/3.5) end to end through Phase 4's post-mutation validation
// (tasks.md 4.3): every use'd module's go.mod, plus go.work's own go
// directive, are mutated, each module is validated in turn inside the
// "golang:"+targetVersion container, and the report names every module's
// outcome plus the "build" validation kind (design.md D-6, tasks.md 4.5).
// This is also the test tasks.md 3.6's regression-guard verification step
// names explicitly (`go test -run TestUpgrade_Workspace`).
func TestUpgrade_Workspace(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "workspace-3-modules")
	withMockDirectory(t, mockDir)

	afterWork := &daggerkit.MockDaggerDirectory{}
	afterVersion := &daggerkit.MockDaggerDirectory{}
	afterA := &daggerkit.MockDaggerDirectory{}
	afterB := &daggerkit.MockDaggerDirectory{}
	afterC := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
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

	modAFile := &daggerkit.MockDaggerFile{}
	modAFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)
	modBFile := &daggerkit.MockDaggerFile{}
	modBFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)
	modCFile := &daggerkit.MockDaggerFile{}
	modCFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	// Every module's pre-tidy go.sum (read from the original, unmutated
	// source directory) differs from its post-tidy go.sum below, so
	// GoSumChanged is driven by a real byte difference — not by
	// AddedModules/RemovedModules, which validatedGoMod's require
	// directive also happens to populate for this fixture.
	mockDir.On("File", "modA/go.sum").Return(mockFileContents(unchangedGoSum))
	mockDir.On("File", "modB/go.sum").Return(mockFileContents(unchangedGoSum))
	mockDir.On("File", "modC/go.sum").Return(mockFileContents(unchangedGoSum))

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterC).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)

	mockContainer.On("WithWorkdir", "/workspace/modA").Return(mockContainer)
	mockContainer.On("File", "/workspace/modA/go.mod").Return(modAFile)
	mockContainer.On("File", "/workspace/modA/go.sum").Return(mockFileContents(changedGoSum))
	mockContainer.On("WithWorkdir", "/workspace/modB").Return(mockContainer)
	mockContainer.On("File", "/workspace/modB/go.mod").Return(modBFile)
	mockContainer.On("File", "/workspace/modB/go.sum").Return(mockFileContents(changedGoSum))
	mockContainer.On("WithWorkdir", "/workspace/modC").Return(mockContainer)
	mockContainer.On("File", "/workspace/modC/go.mod").Return(modCFile)
	mockContainer.On("File", "/workspace/modC/go.sum").Return(mockFileContents(changedGoSum))

	mockContainer.On("Directory", "/workspace").Return(finalDir)
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

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
	assert.Equal(t, "build", report.Validation)
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
		// validatedGoMod's require directive is absent from every
		// fixture module's own (pre-tidy) go.mod, so every module's
		// delta reports it added.
		assert.Equal(t, []string{"example.com/dep"}, m.AddedModules)
		assert.Empty(t, m.RemovedModules)
		// GoSumChanged is driven by the mocked go.sum byte difference
		// above (unchangedGoSum before, changedGoSum after), not by the
		// require-list delta.
		assert.True(t, m.GoSumChanged)
	}

	mockDir.AssertExpectations(t)
	afterWork.AssertExpectations(t)
	afterVersion.AssertExpectations(t)
	afterA.AssertExpectations(t)
	afterB.AssertExpectations(t)
	afterC.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
}

// TestGoRuntimeUpgrader_Upgrade_WritesUpgradeReport proves the report
// content written to runtimeUpgradeReportPath matches
// shipwright.UpgradeReport's contract, including Phase 4's Validation
// field (design.md D-6, tasks.md 4.5), the per-module require-list delta
// fields, and GoSumChanged's real go.sum byte comparison (design.md D-7,
// tasks.md 4.4).
func TestGoRuntimeUpgrader_Upgrade_WritesUpgradeReport(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	afterModFile := &daggerkit.MockDaggerFile{}
	afterModFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	mockDir.On("File", "go.sum").Return(mockFileContents(unchangedGoSum))

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileContents(changedGoSum))
	mockContainer.On("Directory", "/workspace").Return(finalDir)

	var capturedReport string
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
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
	assert.Equal(t, "build", report.Validation)
	require.Len(t, report.Modules, 1)
	assert.Equal(t, ".", report.Modules[0].Path)
	assert.Equal(t, "1.26.7", report.Modules[0].PreviousGo)
	assert.Equal(t, "1.27.0", report.Modules[0].UpdatedGo)
	// GoSumChanged is driven by the mocked go.sum byte difference above
	// (unchangedGoSum before, changedGoSum after), not by the
	// require-list delta below.
	assert.True(t, report.Modules[0].GoSumChanged)
	assert.Equal(t, []string{"example.com/dep"}, report.Modules[0].AddedModules)
	assert.Empty(t, report.Modules[0].RemovedModules)
}

// TestGoRuntimeUpgrader_Upgrade_ValidationFailure_ReturnsNilDirectory is
// tasks.md 4.1's RED test: a `go build ./...` failure inside the
// validation container returns (nil, err) — a *ValidationError describing
// which module and which validation stage failed — never a directory
// presented as a successfully upgraded workspace (spec: "Post-mutation
// validation failure is not silently returned").
func TestGoRuntimeUpgrader_Upgrade_ValidationFailure_ReturnsNilDirectory(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	execErr := errors.New(`process "go build ./..." did not complete successfully: exit code 2: stderr: ./main.go:3:2: undefined: foo`)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, execErr)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "build", validationErr.Validation)
	assert.Equal(t, ".", validationErr.Failed)
	assert.Empty(t, validationErr.Succeeded)
	assert.Contains(t, err.Error(), "stderr: ./main.go:3:2: undefined: foo")
	require.ErrorIs(t, err, execErr)

	// No directory is ever exported once validation has failed: the
	// mutated tree is never presented as if it were a clean upgrade.
	mockContainer.AssertNotCalled(t, "Directory", mock.Anything)
	mockContainer.AssertNotCalled(t, "File", mock.Anything)
}

// TestGoRuntimeUpgrader_Upgrade_TidyFailure_ClassifiesAsTidy proves
// ValidationError.Validation names "tidy", not the generic "build", when
// the failing Sync's error names `go mod tidy` — Dagger's own ExecError
// (and the buildkit process error it wraps) renders as `process "<argv>"
// did not complete successfully: ...`, naming the exact failing command,
// which classifyValidationFailure matches on.
func TestGoRuntimeUpgrader_Upgrade_TidyFailure_ClassifiesAsTidy(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	execErr := errors.New(`process "go mod tidy" did not complete successfully: exit code 1: stderr: go: example.com/dep@v1.0.0: reading example.com/dep/go.mod at revision v1.0.0: unrecognized import path`)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "mod", "tidy"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, execErr)

	upgrader := &GoRuntimeUpgrader{Client: mockClient, Tidy: true}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "tidy", validationErr.Validation)
	assert.Equal(t, ".", validationErr.Failed)
}

// TestGoRuntimeUpgrader_Upgrade_ToolchainImageFailure_ClassifiesAsToolchain
// proves ValidationError.Validation names "toolchain", not "build", when
// the failing Sync's error names neither `go mod tidy` nor `go build
// ./...` — the shape a "golang:"+targetVersion image that can't be
// resolved actually takes, since Container().From is lazy and only
// surfaces its own failure at Sync, before any WithExec ever runs.
func TestGoRuntimeUpgrader_Upgrade_ToolchainImageFailure_ClassifiesAsToolchain(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)

	execErr := errors.New(`failed to resolve image reference "docker.io/library/golang:1.27.0": not found`)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, execErr)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "toolchain", validationErr.Validation)
	assert.Equal(t, ".", validationErr.Failed)
}

// TestGoRuntimeUpgrader_Upgrade_OneModuleFailsValidation_FailsWholeOperation
// is tasks.md 4.2's RED test (spec: "One module's validation failure
// scenario"): a go.work referencing two-plus modules where one fails
// post-mutation validation must fail the whole operation, and the error
// must name which module failed and which module(s) already succeeded —
// modC, unreached because modB already aborted the operation, is proven
// never even attempted.
func TestGoRuntimeUpgrader_Upgrade_OneModuleFailsValidation_FailsWholeOperation(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "workspace-3-modules")
	withMockDirectory(t, mockDir)

	afterWork := &daggerkit.MockDaggerDirectory{}
	afterVersion := &daggerkit.MockDaggerDirectory{}
	afterA := &daggerkit.MockDaggerDirectory{}
	afterB := &daggerkit.MockDaggerDirectory{}
	afterC := &daggerkit.MockDaggerDirectory{}

	mockDir.On("WithNewFile", "go.work", mock.AnythingOfType("string")).Return(afterWork)
	afterWork.On("WithNewFile", ".go-version", mock.AnythingOfType("string")).Return(afterVersion)
	afterVersion.On("WithNewFile", "modA/go.mod", mock.AnythingOfType("string")).Return(afterA)
	afterA.On("WithNewFile", "modB/go.mod", mock.AnythingOfType("string")).Return(afterB)
	afterB.On("WithNewFile", "modC/go.mod", mock.AnythingOfType("string")).Return(afterC)

	modAFile := &daggerkit.MockDaggerFile{}
	modAFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	// modA is the only module that reaches recordModuleDelta (modB fails
	// validation before it gets there, modC is never attempted).
	mockDir.On("File", "modA/go.sum").Return(mockFileContents(unchangedGoSum))

	execErr := errors.New(`process "go build ./..." did not complete successfully: exit code 2: stderr: modB does not compile`)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterC).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)

	mockContainer.On("WithWorkdir", "/workspace/modA").Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil).Once()
	mockContainer.On("File", "/workspace/modA/go.mod").Return(modAFile)
	mockContainer.On("File", "/workspace/modA/go.sum").Return(mockFileContents(unchangedGoSum))

	mockContainer.On("WithWorkdir", "/workspace/modB").Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, execErr).Once()

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")

	require.Error(t, err)
	assert.Nil(t, out)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "build", validationErr.Validation)
	assert.Equal(t, "modB", validationErr.Failed)
	assert.Equal(t, []string{"modA"}, validationErr.Succeeded)

	mockContainer.AssertNotCalled(t, "WithWorkdir", "/workspace/modC")
	mockContainer.AssertNotCalled(t, "File", "/workspace/modB/go.mod")
	mockContainer.AssertNotCalled(t, "Directory", mock.Anything)
}

// TestGoRuntimeUpgrader_Upgrade_GoSumChanged_VersionOnlyBump_DetectsChange
// proves the false-negative bug fix: `go mod tidy` bumping an *existing*
// dependency's version leaves the go.mod require-list path set unchanged
// (only the pinned version differs, which requireModulePaths never looks
// at) while go.sum's hash entries genuinely change. The old require-path-
// diff proxy reported GoSumChanged == false for exactly this scenario;
// the real go.sum byte comparison must report true.
func TestGoRuntimeUpgrader_Upgrade_GoSumChanged_VersionOnlyBump_DetectsChange(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module-dep-bump")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)
	mockDir.On("File", "go.sum").Return(mockFileContents(unchangedGoSum))

	// Same require path as the fixture's go.mod (example.com/dep), only
	// the pinned version differs — a realistic `go mod tidy` version
	// bump, never a require added/removed.
	const versionBumpedGoMod = "module example.com/fixture/single-dep-bump\n\ngo 1.27.0\n\nrequire example.com/dep v1.0.1\n"
	afterModFile := mockFileContents(versionBumpedGoMod)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileContents(changedGoSum))
	mockContainer.On("Directory", "/workspace").Return(finalDir)

	var capturedReport string
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)

	var report shipwright.UpgradeReport
	require.NoError(t, json.Unmarshal([]byte(capturedReport), &report))
	require.Len(t, report.Modules, 1)
	assert.Empty(t, report.Modules[0].AddedModules, "require path set is unchanged — only the pinned version differs")
	assert.Empty(t, report.Modules[0].RemovedModules)
	assert.True(t, report.Modules[0].GoSumChanged, "go.sum content differs even though the require path set did not")
}

// TestGoRuntimeUpgrader_Upgrade_GoSumUnchanged_ReportsFalse proves
// GoSumChanged stays false when go.sum's content is genuinely identical
// before and after `go mod tidy` ran.
func TestGoRuntimeUpgrader_Upgrade_GoSumUnchanged_ReportsFalse(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)
	mockDir.On("File", "go.sum").Return(mockFileContents(unchangedGoSum))

	afterModFile := &daggerkit.MockDaggerFile{}
	afterModFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileContents(unchangedGoSum))
	mockContainer.On("Directory", "/workspace").Return(finalDir)

	var capturedReport string
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)

	var report shipwright.UpgradeReport
	require.NoError(t, json.Unmarshal([]byte(capturedReport), &report))
	require.Len(t, report.Modules, 1)
	assert.False(t, report.Modules[0].GoSumChanged)
}

// TestGoRuntimeUpgrader_Upgrade_GoSumAbsentBeforeAndAfter_ReportsFalse
// proves a module with no go.sum on either side of validation (e.g. zero
// external dependencies) reports GoSumChanged == false rather than
// treating the missing file as a hard failure.
func TestGoRuntimeUpgrader_Upgrade_GoSumAbsentBeforeAndAfter_ReportsFalse(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockDir := mockDirFromFixture(t, "single-module")
	withMockDirectory(t, mockDir)

	afterMod := &daggerkit.MockDaggerDirectory{}
	finalDir := &daggerkit.MockDaggerDirectory{}
	realDir := &dagger.Directory{}

	mockDir.On("WithNewFile", "go.mod", mock.AnythingOfType("string")).Return(afterMod)
	mockDir.On("File", "go.sum").Return(mockFileNotFound())

	afterModFile := &daggerkit.MockDaggerFile{}
	afterModFile.On("Contents", mock.Anything).Return(validatedGoMod, nil)

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.27.0").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/workspace", afterMod).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/workspace").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("File", "/workspace/go.mod").Return(afterModFile)
	mockContainer.On("File", "/workspace/go.sum").Return(mockFileNotFound())
	mockContainer.On("Directory", "/workspace").Return(finalDir)

	var capturedReport string
	finalDir.On("WithNewFile", ".shipwright/runtime-upgrade-report.json", mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		capturedReport = args.String(1)
	}).Return(finalDir)
	finalDir.On("GetRealDirectory").Return(realDir)

	upgrader := &GoRuntimeUpgrader{Client: mockClient}

	out, err := upgrader.Upgrade(context.Background(), &dagger.Directory{}, "1.27.0")
	require.NoError(t, err)
	assert.Same(t, realDir, out)

	var report shipwright.UpgradeReport
	require.NoError(t, json.Unmarshal([]byte(capturedReport), &report))
	require.Len(t, report.Modules, 1)
	assert.False(t, report.Modules[0].GoSumChanged)
}
