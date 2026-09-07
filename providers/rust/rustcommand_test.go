package rust_test

import (
	"context"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

func TestRustCommand_Test_NilClient(t *testing.T) {
	command := &rust.RustCommand{Command: "run -p xtask -- verify-layers"}

	out, err := command.Test(context.Background(), nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "dagger client is not configured")
	require.Nil(t, out)
}

func TestRustCommand_Test_NilSource(t *testing.T) {
	command := &rust.RustCommand{
		Client:  &daggerkit.MockDaggerClient{},
		Command: "run -p xtask -- verify-layers",
	}

	out, err := command.Test(context.Background(), nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "source directory is nil")
	require.Nil(t, out)
}

func TestRustCommand_Test_RequiresCommand(t *testing.T) {
	command := &rust.RustCommand{Client: &daggerkit.MockDaggerClient{}}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "command is required")
	require.Nil(t, out)
}

// mockCommandContainer builds the mocked container chain shared by the
// mock-client tests below, parameterized by the WithExec argv the test
// expects RustCommand to have built and the (stdout, err) the exec should
// report.
func mockCommandContainer(t *testing.T, wantExecArgs []string, stdout string, execErr error) *daggerkit.MockDaggerClient {
	t.Helper()

	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", wantExecArgs).Return(container)
	container.On("Stdout", mock.Anything).Return(stdout, execErr)

	if execErr == nil {
		realFile := &dagger.File{}
		reportFile := &daggerkit.MockDaggerFile{}
		reportFile.On("GetRealFile").Return(realFile)
		container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
		container.On("File", mock.Anything).Return(reportFile)
	}

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	return client
}

// TestRustCommand_Test_MockClient_XtaskInvocation is the first validation
// stage the ego-rs migration calls for: an innocuous `cargo run -p xtask --
// verify-layers` with no ManifestPath and Docker left false must resolve to
// exactly that argv, with no Docker-in-Docker daemon attached.
func TestRustCommand_Test_MockClient_XtaskInvocation(t *testing.T) {
	client := mockCommandContainer(t, []string{"cargo", "run", "-p", "xtask", "--", "verify-layers"}, "verify-layers: ok\n", nil)

	command := &rust.RustCommand{
		Client:  client,
		Command: "run -p xtask -- verify-layers",
	}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.NotNil(t, out)
	client.AssertNotCalled(t, "Host")
}

// TestRustCommand_Test_MockClient_NonZeroExitPropagates covers the
// pass/fail contract: a failing cargo invocation (non-zero exit, surfaced by
// Dagger as an error from Stdout) must fail Test, never a partial/inferred
// success.
func TestRustCommand_Test_MockClient_NonZeroExitPropagates(t *testing.T) {
	execErr := &dagger.ExecError{
		ExitCode: 1,
		Stderr:   "thread 'main' panicked at verify-layers\n",
	}
	client := mockCommandContainer(t, []string{"cargo", "run", "-p", "xtask", "--", "verify-layers"}, "", execErr)

	command := &rust.RustCommand{
		Client:  client,
		Command: "run -p xtask -- verify-layers",
	}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "exit code 1")
	require.Contains(t, err.Error(), "panicked at verify-layers")
	require.Nil(t, out)
}

// TestRustCommand_Test_MockClient_ManifestPathAppliedConsistently covers the
// duplication concern the ego-rs manifest review raised: a ManifestPath
// with-field must become --manifest-path in the actual argv, not something
// the caller has to also embed inside Command.
func TestRustCommand_Test_MockClient_ManifestPathAppliedConsistently(t *testing.T) {
	client := mockCommandContainer(t,
		[]string{"cargo", "run", "--manifest-path", "integration-tests/Cargo.toml", "--bin", "run-suite"},
		"run-suite: ok\n", nil)

	command := &rust.RustCommand{
		Client:       client,
		ManifestPath: "integration-tests/Cargo.toml",
		Command:      "run --bin run-suite",
	}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.NotNil(t, out)
}

// TestRustCommand_Test_MockClient_DockerAbsentByDefault covers the safety
// default: xtask-style commands must never get a Docker-in-Docker daemon
// attached unless Docker is explicitly set to true.
func TestRustCommand_Test_MockClient_DockerAbsentByDefault(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", []string{"cargo", "run", "-p", "xtask", "--", "verify-layers"}).Return(container)
	container.On("Stdout", mock.Anything).Return("ok\n", nil)

	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	command := &rust.RustCommand{
		Client:  client,
		Command: "run -p xtask -- verify-layers",
	}

	_, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	container.AssertNotCalled(t, "AsService")
	container.AssertNotCalled(t, "WithServiceBinding", mock.Anything, mock.Anything)
	container.AssertNotCalled(t, "WithEnvVariable", mock.Anything, mock.Anything)
}

// TestRustCommand_Test_MockClient_DockerAttachesDinDWhenConfigured covers
// run-suite's own requirement: setting Docker: true must attach a real
// Docker-in-Docker daemon (see dockerdaemon.go) via WithServiceBinding and
// point DOCKER_HOST at its network alias, replacing the earlier
// Docker-outside-of-Docker (mounted host socket) mechanism, which cannot
// work through Dagger's nested container execution model — see
// dockerdaemon.go's doc comment for the full root cause.
func TestRustCommand_Test_MockClient_DockerAttachesDinDWhenConfigured(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExposedPort", 2375).Return(container)
	container.On("WithInsecureExec", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}).Return(container)

	dindService := &daggerkit.MockDaggerService{}
	container.On("AsService").Return(dindService)
	container.On("WithServiceBinding", "docker", dindService).Return(container)
	container.On("WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375").Return(container)

	container.On("WithExec", []string{"cargo", "run", "--manifest-path", "integration-tests/Cargo.toml", "--bin", "run-suite"}).Return(container)
	container.On("Stdout", mock.Anything).Return("run-suite: ok\n", nil)

	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	command := &rust.RustCommand{
		Client:       client,
		ManifestPath: "integration-tests/Cargo.toml",
		Command:      "run --bin run-suite",
		Docker:       true,
	}

	out, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.NotNil(t, out)
	container.AssertCalled(t, "AsService")
	container.AssertCalled(t, "WithServiceBinding", "docker", dindService)
	container.AssertCalled(t, "WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375")
}

// TestRustCommand_Test_MockClient_DefaultCacheKey and
// TestRustCommand_Test_MockClient_ExplicitCacheKey cover CacheKey isolation:
// an unset CacheKey must mount the shared default target cache volume, and
// an explicit one must mount its own namespaced volume — so two RustCommand
// steps against different Cargo workspaces never share incompatible
// incremental-compilation state.
func TestRustCommand_Test_MockClient_DefaultCacheKey(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", mock.Anything).Return(container)
	container.On("Stdout", mock.Anything).Return("ok\n", nil)
	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	registryCache := &daggerkit.MockDaggerCacheVolume{}
	targetCache := &daggerkit.MockDaggerCacheVolume{}
	container.On("WithMountedCache", "/usr/local/cargo/registry", registryCache).Return(container)
	container.On("WithMountedCache", "/src/target", targetCache).Return(container)

	client.On("Container").Return(container)
	client.On("CacheVolume", "shipwright-cargo-registry").Return(registryCache)
	client.On("CacheVolume", "shipwright-rust-command-target").Return(targetCache)

	command := &rust.RustCommand{Client: client, Command: "run -p xtask -- verify-layers"}

	_, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	client.AssertCalled(t, "CacheVolume", "shipwright-rust-command-target")
}

func TestRustCommand_Test_MockClient_ExplicitCacheKey(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", mock.Anything).Return(container)
	container.On("Stdout", mock.Anything).Return("ok\n", nil)
	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	registryCache := &daggerkit.MockDaggerCacheVolume{}
	targetCache := &daggerkit.MockDaggerCacheVolume{}
	container.On("WithMountedCache", "/usr/local/cargo/registry", registryCache).Return(container)
	container.On("WithMountedCache", "/src/target", targetCache).Return(container)

	client.On("Container").Return(container)
	client.On("CacheVolume", "shipwright-cargo-registry").Return(registryCache)
	client.On("CacheVolume", "shipwright-rust-command-target-run-suite").Return(targetCache)

	command := &rust.RustCommand{
		Client:   client,
		Command:  "run --manifest-path integration-tests/Cargo.toml --bin run-suite",
		CacheKey: "run-suite",
	}

	_, err := command.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	client.AssertCalled(t, "CacheVolume", "shipwright-rust-command-target-run-suite")
}
