package golang_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestGoIntegrationTester_Test_NilClient(t *testing.T) {
	tester := &golang.GoIntegrationTester{}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	if err == nil {
		t.Fatal("GoIntegrationTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoIntegrationTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoIntegrationTester.Test() file = %v, want nil on error", out)
	}
}

func TestGoIntegrationTester_Test_NilSource(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	tester := &golang.GoIntegrationTester{Client: mockClient}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoIntegrationTester.Test() error = nil, want error for nil source directory")
	}
	if !strings.Contains(err.Error(), "source directory is nil") {
		t.Fatalf("GoIntegrationTester.Test() error = %v, want it to mention a nil source directory", err)
	}
	if out != nil {
		t.Fatalf("GoIntegrationTester.Test() file = %v, want nil on error", out)
	}
}

// goIntegrationTesterFixture wires every mock expectation the DinD wiring +
// caches + env chain requires, shared by every success-path test below.
// client.Container() is called twice by the production code — once for the
// GoIntegrationTester's own container, once inside withDockerDaemon for the
// DinD sidecar — and both calls resolve to the SAME mockContainer instance,
// so a single set of precise (non mock.Anything) `From(...)` expectations
// covers both call sites, mirroring dockerdaemon_test.go's own wiring
// assertions rather than rustintegrationtester_test.go's looser
// mock.Anything shortcut.
type goIntegrationTesterFixture struct {
	client         *daggerkit.MockDaggerClient
	container      *daggerkit.MockDaggerContainer
	dindService    *daggerkit.MockDaggerService
	reportFile     *daggerkit.MockDaggerFile
	realReportFile *dagger.File
}

func newGoIntegrationTesterFixture(goVersion string) *goIntegrationTesterFixture {
	f := &goIntegrationTesterFixture{
		client:         &daggerkit.MockDaggerClient{},
		container:      &daggerkit.MockDaggerContainer{},
		dindService:    &daggerkit.MockDaggerService{},
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

	// withDockerDaemon's own wiring (already pinned in detail by
	// dockerdaemon_test.go; asserted here only to prove GoIntegrationTester
	// actually calls it, D-9/O2).
	f.container.On("From", "docker:27-dind").Return(f.container)
	f.container.On("WithExposedPort", 2375).Return(f.container)
	f.container.On("WithEnvVariable", "SHIPWRIGHT_DIND_INSTANCE", mock.AnythingOfType("string")).Return(f.container)
	f.container.On("AsService", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}).Return(f.dindService)
	f.container.On("WithServiceBinding", "docker", f.dindService).Return(f.container)
	f.container.On("WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375").Return(f.container)

	f.container.On("WithEnvVariable", "TESTCONTAINERS_RYUK_DISABLED", "true").Return(f.container)

	return f
}

// TestGoIntegrationTester_Test_DefaultCommand pins D-2: an empty Command
// resolves to the exact default `go test -tags=integration ./...`, run via
// `sh -c` (D-3), and the successful run's stdout becomes the report file's
// body verbatim.
func TestGoIntegrationTester_Test_DefaultCommand(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.26.7")
	const testOutput = "ok  \tdindfixture\t0.42s\n"

	f.container.On("WithExec", []string{"sh", "-c", "go test -tags=integration ./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return(testOutput, nil)
	f.container.On("WithNewFile", "/tmp/integration-test-output.txt", testOutput).Return(f.container)
	f.container.On("File", "/tmp/integration-test-output.txt").Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)

	tester := &golang.GoIntegrationTester{Client: f.client}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertExpectations(t)
}

// TestGoIntegrationTester_Test_CommandOverride pins D-3, the whole point of
// O1: a Command containing `&&` is passed to `sh -c` as a single argv
// element, never tokenized or rejected — the exact invocation
// `rust-command`'s whitespace-argv approach cannot express.
func TestGoIntegrationTester_Test_CommandOverride(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.26.7")
	const command = "go build && ./inttest-runner"
	const testOutput = "runner: all good\n"

	f.container.On("WithExec", []string{"sh", "-c", command}, daggerkit.DaggerContainerWithExecOpts{}).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return(testOutput, nil)
	f.container.On("WithNewFile", "/tmp/integration-test-output.txt", testOutput).Return(f.container)
	f.container.On("File", "/tmp/integration-test-output.txt").Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)

	tester := &golang.GoIntegrationTester{Client: f.client, Command: command}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertExpectations(t)
}

// TestGoIntegrationTester_Test_DockerAlwaysAttached pins D-9/O2: there is no
// config path that omits withDockerDaemon's wiring — Docker access is
// inherent to this capability, mirroring RustIntegrationTester's own
// unconditional attachment.
func TestGoIntegrationTester_Test_DockerAlwaysAttached(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.26.7")

	f.container.On("WithExec", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return("ok\n", nil)
	f.container.On("WithNewFile", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("File", mock.Anything).Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)

	tester := &golang.GoIntegrationTester{Client: f.client}

	_, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	f.container.AssertCalled(t, "AsService", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"})
	f.container.AssertCalled(t, "WithServiceBinding", "docker", f.dindService)
	f.container.AssertCalled(t, "WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375")
}

// TestGoIntegrationTester_Test_CachesAndEnv pins D-7/D-8/D-9: both shared Go
// caches are mounted, GO111MODULE=on is set, CGO_ENABLED is never touched
// (an arbitrary Command may need cgo on, e.g. `-race`, or off for a static
// binary — GoIntegrationTester must not decide for it), and
// TESTCONTAINERS_RYUK_DISABLED=true is set unconditionally (D-9, a provider
// default pending real-run evidence, overridable from inside Command).
func TestGoIntegrationTester_Test_CachesAndEnv(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.26.7")

	f.container.On("WithExec", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return("ok\n", nil)
	f.container.On("WithNewFile", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("File", mock.Anything).Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)

	tester := &golang.GoIntegrationTester{Client: f.client}

	_, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	f.container.AssertCalled(t, "WithMountedCache", "/go/pkg/mod", mock.Anything)
	f.container.AssertCalled(t, "WithMountedCache", "/root/.cache/go-build", mock.Anything)
	f.container.AssertCalled(t, "WithEnvVariable", "GO111MODULE", "on")
	f.container.AssertCalled(t, "WithEnvVariable", "TESTCONTAINERS_RYUK_DISABLED", "true")
	f.container.AssertNotCalled(t, "WithEnvVariable", "CGO_ENABLED", mock.Anything)
}

// TestGoIntegrationTester_Test_Failure proves a non-zero integration run
// surfaces as a wrapped "gointegrationtester:" error (Dagger's ExecError
// already embeds stdout+stderr, design.md D-10), and that no report file is
// ever written for a failed run.
func TestGoIntegrationTester_Test_Failure(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.26.7")
	testErr := errors.New("exit code 1: --- FAIL: TestSomething (0.01s)")

	f.container.On("WithExec", []string{"sh", "-c", "go test -tags=integration ./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return("", testErr)

	tester := &golang.GoIntegrationTester{Client: f.client}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gointegrationtester: integration tests failed")
	assert.Contains(t, err.Error(), "TestSomething")
	assert.Nil(t, out)
	f.container.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}

// TestGoIntegrationTester_Test_GoVersionOverride pins design.md D-11:
// GoVersion actually selects the toolchain image, same convention as
// GoBuildChecker.GoVersion.
func TestGoIntegrationTester_Test_GoVersionOverride(t *testing.T) {
	f := newGoIntegrationTesterFixture("1.25.5")

	f.container.On("WithExec", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("Stdout", mock.Anything).Return("ok\n", nil)
	f.container.On("WithNewFile", mock.Anything, mock.Anything).Return(f.container)
	f.container.On("File", mock.Anything).Return(f.reportFile)
	f.reportFile.On("GetRealFile").Return(f.realReportFile)

	tester := &golang.GoIntegrationTester{Client: f.client, GoVersion: "1.25.5"}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	assert.Same(t, f.realReportFile, out)
	f.container.AssertCalled(t, "From", "golang:1.25.5")
}
