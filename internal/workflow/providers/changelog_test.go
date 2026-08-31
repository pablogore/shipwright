package providers_test

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/internal/daggerkit"
	"github.com/pablogore/shipwright/internal/workflow/providers"
)

// changelogWithExecOpts is the zero-value DaggerContainerWithExecOpts every
// WithExec call in changelog.go's Run is built with — none of its WithExec
// invocations set RedirectStdout.
var changelogWithExecOpts = daggerkit.DaggerContainerWithExecOpts{}

// changelogApkAddArgs and changelogSafeDirectoryArgs are the two fixed
// WithExec argument slices ChangelogRunner.Run issues before ever branching
// on tag/log output, shared by every mock-based test below.
var (
	changelogApkAddArgs        = []string{"apk", "add", "--no-cache", "git"}
	changelogSafeDirectoryArgs = []string{"git", "config", "--global", "--add", "safe.directory", "/work"}
	changelogDescribeArgs      = []string{"git", "describe", "--tags", "--abbrev=0"}
)

// newChangelogBaseContainerMock wires up the container chain common to
// every ChangelogRunner.Run call that reaches past the nil-client/nil-source
// guards: Client.Container().From(image).WithMountedDirectory(...).
// WithWorkdir(...).WithExec(apk add git).WithExec(safe.directory), returning
// the mock client and the mock container the chain settles on (matching
// container.go's own "container" local variable).
func newChangelogBaseContainerMock(t *testing.T, build *dagger.Directory) (*daggerkit.MockDaggerClient, *daggerkit.MockDaggerContainer) {
	t.Helper()

	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "alpine:3.24.1").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/work", mock.MatchedBy(func(d daggerkit.DaggerDirectory) bool {
		return d.GetRealDirectory() == build
	})).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/work").Return(mockContainer)
	mockContainer.On("WithExec", changelogApkAddArgs, changelogWithExecOpts).Return(mockContainer)
	mockContainer.On("WithExec", changelogSafeDirectoryArgs, changelogWithExecOpts).Return(mockContainer)

	return mockClient, mockContainer
}

// TestChangelogRunner_Run_NilSource_Mock is the mock-based counterpart of
// TestChangelogRunner_Run_NilSource_RealClient: it proves the nil-source
// guard fires before any Dagger call is made, using a mock client that
// would fail the test (via AssertExpectations/AssertNotCalled) if Run ever
// reached into it.
func TestChangelogRunner_Run_NilSource_Mock(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	runner := &providers.ChangelogRunner{Client: mockClient}

	out, err := runner.Run(context.Background(), nil)

	require.Error(t, err)
	assert.Nil(t, out)
	mockClient.AssertNotCalled(t, "Container")
}

// TestChangelogRunner_Run_EmptyHistoryLeavesChangelogUntouched_Mock is the
// mock-based counterpart of
// TestChangelogRunner_Run_RealEngine_EmptyHistoryLeavesChangelogUntouched:
// both `git describe` and the full-history `git log` fail (an unborn HEAD),
// so Run must fall through to "no classifiable commits" and hand back the
// container completely untouched -- no File/WithNewFile call at all.
func TestChangelogRunner_Run_EmptyHistoryLeavesChangelogUntouched_Mock(t *testing.T) {
	build := &dagger.Directory{}
	mockClient, mockContainer := newChangelogBaseContainerMock(t, build)

	describeContainer := &daggerkit.MockDaggerContainer{}
	mockContainer.On("WithExec", changelogDescribeArgs, changelogWithExecOpts).Return(describeContainer)
	describeContainer.On("Stdout", mock.Anything).Return("", errors.New("fatal: No names found, cannot describe anything"))

	logContainer := &daggerkit.MockDaggerContainer{}
	fullHistoryLogArgs := []string{"git", "log", "--pretty=format:%s"}
	mockContainer.On("WithExec", fullHistoryLogArgs, changelogWithExecOpts).Return(logContainer)
	logContainer.On("Stdout", mock.Anything).Return("", errors.New("fatal: your current branch does not have any commits yet"))

	realContainer := &dagger.Container{}
	mockContainer.On("GetRealContainer").Return(realContainer)

	runner := &providers.ChangelogRunner{Client: mockClient}
	out, err := runner.Run(context.Background(), build)

	require.NoError(t, err)
	assert.Same(t, realContainer, out)
	mockContainer.AssertNotCalled(t, "File", mock.Anything)
	mockContainer.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
	mockClient.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
	describeContainer.AssertExpectations(t)
	logContainer.AssertExpectations(t)
}

// TestChangelogRunner_Run_Success_Mock is the mock-based counterpart of
// TestChangelogRunner_Run_RealEngine: a reachable tag plus feat/fix commits
// since that tag classify into Added/Fixed sections that get prepended
// ahead of the pre-existing changelog content, under the "## [Unreleased]"
// heading.
func TestChangelogRunner_Run_Success_Mock(t *testing.T) {
	build := &dagger.Directory{}
	mockClient, mockContainer := newChangelogBaseContainerMock(t, build)

	describeContainer := &daggerkit.MockDaggerContainer{}
	mockContainer.On("WithExec", changelogDescribeArgs, changelogWithExecOpts).Return(describeContainer)
	describeContainer.On("Stdout", mock.Anything).Return("v1.2.3\n", nil)

	logContainer := &daggerkit.MockDaggerContainer{}
	sinceTagLogArgs := []string{"git", "log", "v1.2.3..HEAD", "--pretty=format:%s"}
	mockContainer.On("WithExec", sinceTagLogArgs, changelogWithExecOpts).Return(logContainer)
	logContainer.On("Stdout", mock.Anything).Return("feat: add first feature\nfix: correct a bug", nil)

	mockFile := &daggerkit.MockDaggerFile{}
	mockContainer.On("File", "CHANGELOG.md").Return(mockFile)
	mockFile.On("Contents", mock.Anything).Return("# Changelog\n\n## [Unreleased]\n\n### Added\n- existing entry\n", nil)

	finalContainer := &daggerkit.MockDaggerContainer{}
	realContainer := &dagger.Container{}
	mockContainer.On("WithNewFile", "CHANGELOG.md", mock.MatchedBy(func(content string) bool {
		return changelogTestContains(content, "### Added\n- add first feature") &&
			changelogTestContains(content, "### Fixed\n- correct a bug") &&
			changelogTestContains(content, "existing entry")
	})).Return(finalContainer)
	finalContainer.On("GetRealContainer").Return(realContainer)

	runner := &providers.ChangelogRunner{Client: mockClient}
	out, err := runner.Run(context.Background(), build)

	require.NoError(t, err)
	assert.Same(t, realContainer, out)
	mockClient.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
	describeContainer.AssertExpectations(t)
	logContainer.AssertExpectations(t)
	mockFile.AssertExpectations(t)
	finalContainer.AssertExpectations(t)
}

// TestChangelogRunner_Run_NoTagBootstrap_Mock is the mock-based counterpart
// of TestChangelogRunner_Run_RealEngine_NoTagBootstrap: `git describe` fails
// (no reachable tag yet), so Run falls back to diffing the entire history
// rather than a tag..HEAD range.
func TestChangelogRunner_Run_NoTagBootstrap_Mock(t *testing.T) {
	build := &dagger.Directory{}
	mockClient, mockContainer := newChangelogBaseContainerMock(t, build)

	describeContainer := &daggerkit.MockDaggerContainer{}
	mockContainer.On("WithExec", changelogDescribeArgs, changelogWithExecOpts).Return(describeContainer)
	describeContainer.On("Stdout", mock.Anything).Return("", errors.New("fatal: No names found, cannot describe anything"))

	logContainer := &daggerkit.MockDaggerContainer{}
	fullHistoryLogArgs := []string{"git", "log", "--pretty=format:%s"}
	mockContainer.On("WithExec", fullHistoryLogArgs, changelogWithExecOpts).Return(logContainer)
	logContainer.On("Stdout", mock.Anything).Return("feat: bootstrap feature", nil)

	mockFile := &daggerkit.MockDaggerFile{}
	mockContainer.On("File", "CHANGELOG.md").Return(mockFile)
	mockFile.On("Contents", mock.Anything).Return("", errors.New("no such file"))

	finalContainer := &daggerkit.MockDaggerContainer{}
	realContainer := &dagger.Container{}
	mockContainer.On("WithNewFile", "CHANGELOG.md", mock.MatchedBy(func(content string) bool {
		return changelogTestContains(content, "### Added\n- bootstrap feature")
	})).Return(finalContainer)
	finalContainer.On("GetRealContainer").Return(realContainer)

	runner := &providers.ChangelogRunner{Client: mockClient}
	out, err := runner.Run(context.Background(), build)

	require.NoError(t, err)
	assert.Same(t, realContainer, out)
	mockClient.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
	describeContainer.AssertExpectations(t)
	logContainer.AssertExpectations(t)
	mockFile.AssertExpectations(t)
	finalContainer.AssertExpectations(t)
}

// TestChangelogRunner_Run_NilClient covers the nil-client guard clause —
// a plain unit test, no Dagger engine involved.
func TestChangelogRunner_Run_NilClient(t *testing.T) {
	runner := &providers.ChangelogRunner{}

	_, err := runner.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("ChangelogRunner.Run() error = nil, want error for an unconfigured client")
	}
}

// changelogTestContains reports whether s contains substr. Named
// distinctly from register_test.go's indexOf/containsModulePath helpers
// (same package, providers_test) to keep each test file's helper naming
// self-explanatory.
func changelogTestContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
