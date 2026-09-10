package golang_test

import (
	"context"
	"errors"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	golang "github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestGoCommand_Test_NilClient(t *testing.T) {
	command := &golang.GoCommand{Command: "build ./..."}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "dagger client is not configured")
	require.Nil(t, out)
}

func TestGoCommand_Test_NilSource(t *testing.T) {
	command := &golang.GoCommand{
		Client:  &daggerkit.MockDaggerClient{},
		Command: "build ./...",
	}

	out, err := command.Test(context.Background(), nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "source directory is nil")
	require.Nil(t, out)
}

// TestGoCommand_Test_EmptyCommand covers both an empty string and a
// whitespace-only Command: Test must reject either before any exec, never
// falling back to bare `go`.
func TestGoCommand_Test_EmptyCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "empty string", command: ""},
		{name: "whitespace only", command: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := &golang.GoCommand{
				Client:  &daggerkit.MockDaggerClient{},
				Command: tt.command,
			}

			out, err := command.Test(context.Background(), &dagger.Directory{})

			require.Error(t, err)
			require.Contains(t, err.Error(), "command is required")
			require.Nil(t, out)
		})
	}
}

// goCommandFixture wires every mock expectation the container-wiring chain
// requires, shared by the tests below — mirrors
// gointegrationtester_test.go's own goIntegrationTesterFixture, minus the
// Docker-in-Docker wiring GoCommand deliberately does not attach.
type goCommandFixture struct {
	client         *daggerkit.MockDaggerClient
	container      *daggerkit.MockDaggerContainer
	reportFile     *daggerkit.MockDaggerFile
	realReportFile *dagger.File
}

func newGoCommandFixture(goVersion string) *goCommandFixture {
	f := &goCommandFixture{
		client:         &daggerkit.MockDaggerClient{},
		container:      &daggerkit.MockDaggerContainer{},
		reportFile:     &daggerkit.MockDaggerFile{},
		realReportFile: &dagger.File{},
	}

	f.client.On("Container").Return(f.container)
	f.container.On("From", "golang:"+goVersion).Return(f.container)
	f.container.On("WithMountedDirectory", "/src", mock.Anything).Return(f.container)
	f.container.On("WithWorkdir", "/src").Return(f.container)
	f.container.On("WithEnvVariable", "GO111MODULE", "on").Return(f.container)
	f.client.On("CacheVolume", "shipwright-go-mod-cache").Return(&daggerkit.MockDaggerCacheVolume{})
	f.client.On("CacheVolume", "shipwright-go-build-cache").Return(&daggerkit.MockDaggerCacheVolume{})
	f.container.On("WithMountedCache", "/go/pkg/mod", mock.Anything).Return(f.container)
	f.container.On("WithMountedCache", "/root/.cache/go-build", mock.Anything).Return(f.container)

	return f
}

func (f *goCommandFixture) expectSuccess(wantExecArgs []string, stdout string) {
	f.container.On("WithExec", wantExecArgs, daggerkit.DaggerContainerWithExecOpts{}).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return(stdout, nil)
	f.container.On("WithNewFile", "/tmp/command-output.txt", stdout).Return(f.container)
	f.container.On("File", "/tmp/command-output.txt").Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)
}

// TestGoCommand_Test_ContainerWiring pins the container wiring copied from
// gointegrationtester.go minus the Docker-in-Docker daemon: golang:<version>
// base image, /src mount + WithWorkdir("/src"), the shared Go caches,
// GO111MODULE=on, and explicitly NO CGO_ENABLED pin and NO Docker daemon
// wiring — go-integration-test remains the sole Go Tester with Docker
// access.
func TestGoCommand_Test_ContainerWiring(t *testing.T) {
	f := newGoCommandFixture("1.26.7")
	f.expectSuccess([]string{"go", "build", "./..."}, "ok\n")

	command := &golang.GoCommand{Client: f.client, Command: "build ./..."}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertCalled(t, "From", "golang:1.26.7")
	f.container.AssertCalled(t, "WithMountedDirectory", "/src", mock.Anything)
	f.container.AssertCalled(t, "WithWorkdir", "/src")
	f.container.AssertCalled(t, "WithMountedCache", "/go/pkg/mod", mock.Anything)
	f.container.AssertCalled(t, "WithMountedCache", "/root/.cache/go-build", mock.Anything)
	f.container.AssertCalled(t, "WithEnvVariable", "GO111MODULE", "on")
	f.container.AssertNotCalled(t, "WithEnvVariable", "CGO_ENABLED", mock.Anything)
	f.container.AssertNotCalled(t, "AsService", mock.Anything, mock.Anything, mock.Anything)
	f.container.AssertNotCalled(t, "WithServiceBinding", mock.Anything, mock.Anything)
	f.container.AssertNotCalled(t, "WithExposedPort", mock.Anything)
}

// TestGoCommand_Test_GoVersionOverride pins that GoVersion actually selects
// the toolchain image, same convention as every other Tester in this
// package.
func TestGoCommand_Test_GoVersionOverride(t *testing.T) {
	f := newGoCommandFixture("1.25.5")
	f.expectSuccess([]string{"go", "vet", "./..."}, "ok\n")

	command := &golang.GoCommand{Client: f.client, GoVersion: "1.25.5", Command: "vet ./..."}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertCalled(t, "From", "golang:1.25.5")
}

// TestGoCommand_Test_DirectorySelection is the threat-matrix "Directory
// selection" RED test (design.md's Threat Matrix, "Directory selection
// (-C, relative/absolute paths)" row): WorkDir is passed as a single argv
// token, never concatenated into a string, for relative, absolute, and
// `..`-traversal paths alike — and an empty WorkDir adds no -C flag at all.
func TestGoCommand_Test_DirectorySelection(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		want    []string
	}{
		{
			name:    "relative workdir",
			workDir: "services/api",
			want:    []string{"go", "-C", "services/api", "build", "./..."},
		},
		{
			name:    "absolute workdir",
			workDir: "/src/services/api",
			want:    []string{"go", "-C", "/src/services/api", "build", "./..."},
		},
		{
			name:    "dot-dot traversal workdir",
			workDir: "../../etc",
			want:    []string{"go", "-C", "../../etc", "build", "./..."},
		},
		{
			name: "empty workdir adds no -C flag",
			want: []string{"go", "build", "./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newGoCommandFixture("1.26.7")
			f.expectSuccess(tt.want, "ok\n")

			command := &golang.GoCommand{Client: f.client, WorkDir: tt.workDir, Command: "build ./..."}

			out, err := command.Test(context.Background(), &dagger.Directory{})

			require.NoError(t, err)
			assert.Same(t, f.realReportFile, out)
			f.container.AssertCalled(t, "WithExec", tt.want, daggerkit.DaggerContainerWithExecOpts{})
		})
	}
}

// TestGoCommand_Test_ShellMetacharactersInert is the threat-matrix "Shell
// metacharacters in Command" RED test (design.md's Threat Matrix row of the
// same name): a Command containing shell metacharacters (`;`, `&&`, `|`,
// `$VAR`) must reach WithExec as literal, inert argv tokens — proof no
// `sh -c` or any shell ever interprets Command, unlike
// GoIntegrationTester.Command (gointegrationtester.go's own doc comment on
// this exact contrast).
func TestGoCommand_Test_ShellMetacharactersInert(t *testing.T) {
	f := newGoCommandFixture("1.26.7")
	const command = "test ./... && rm -rf /"
	wantArgs := []string{"go", "test", "./...", "&&", "rm", "-rf", "/"}
	f.expectSuccess(wantArgs, "ok\n")

	goCommand := &golang.GoCommand{Client: f.client, Command: command}

	out, err := goCommand.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertCalled(t, "WithExec", wantArgs, daggerkit.DaggerContainerWithExecOpts{})
	f.container.AssertNotCalled(t, "WithExec", []string{"sh", "-c", command}, mock.Anything)
}

// TestGoCommand_Test_Success and TestGoCommand_Test_ExecFailure cover the
// exit-code-exclusive pass/fail contract: a successful invocation returns
// the captured stdout as the report File's body, a failing one returns a
// wrapped error and no File — mirroring rustcommand_test.go's own
// MockClient_XtaskInvocation/NonZeroExitPropagates pair.
func TestGoCommand_Test_Success(t *testing.T) {
	f := newGoCommandFixture("1.26.7")
	const testOutput = "ok  \tgithub.com/pablogore/shipwright/cmd/api\t0.42s\n"
	f.expectSuccess([]string{"go", "test", "./..."}, testOutput)

	command := &golang.GoCommand{Client: f.client, Command: "test ./..."}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
}

// TestGoCommand_Test_ExecFailure proves a non-zero go invocation surfaces
// as a wrapped "gocommand:" error (Dagger's ExecError already embeds
// stdout+stderr in its own Error() text, per gobuildchecker.go's doc
// comment — the mock's error carries pre-rendered text mirroring that
// shape, matching gointegrationtester_test.go's own
// TestGoIntegrationTester_Test_Failure convention, since *dagger.ExecError
// cannot be directly struct-literal-constructed outside a real client
// without panicking its own Error() method), and that no report file is
// ever written for a failed run.
func TestGoCommand_Test_ExecFailure(t *testing.T) {
	f := newGoCommandFixture("1.26.7")
	testErr := errors.New("exit code 1: --- FAIL: TestSomething (0.01s)")
	f.container.On("WithExec", []string{"go", "test", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return("", testErr)

	command := &golang.GoCommand{Client: f.client, Command: "test ./..."}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gocommand: go command failed")
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Contains(t, err.Error(), "TestSomething")
	assert.Nil(t, out)
	require.ErrorIs(t, err, testErr)
	f.container.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}
