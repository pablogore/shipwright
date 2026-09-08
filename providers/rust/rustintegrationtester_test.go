package rust_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

func TestRustIntegrationTester_Test_NilClient(t *testing.T) {
	tester := &rust.RustIntegrationTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustIntegrationTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustIntegrationTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustIntegrationTester.Test() file = %v, want nil on error", out)
	}
}

// TestRustIntegrationTester_Test_MockClient_DockerAttachesDinDUnconditionally
// covers this capability's Docker access as inherent, not opt-in: it must
// always attach a real Docker-in-Docker daemon (see dockerdaemon.go) via
// WithServiceBinding and point DOCKER_HOST at its network alias, replacing
// the earlier Docker-outside-of-Docker (mounted host socket) mechanism,
// which cannot work through Dagger's nested container execution model — see
// dockerdaemon.go's doc comment for the full root cause, discovered
// validating rust-command (which shared this exact DooD mechanism) against
// ego-rs's real run-suite.
func TestRustIntegrationTester_Test_MockClient_DockerAttachesDinDUnconditionally(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExposedPort", 2375).Return(container)

	dindService := &daggerkit.MockDaggerService{}
	container.On("WithEnvVariable", "SHIPWRIGHT_DIND_INSTANCE", mock.Anything).Return(container)
	container.On("AsService", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}).Return(dindService)
	container.On("WithServiceBinding", "docker", dindService).Return(container)
	container.On("WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375").Return(container)

	container.On("WithExec", mock.Anything).Return(container)
	container.On("Stdout", mock.Anything).Return("integration: ok\n", nil)

	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	tester := &rust.RustIntegrationTester{
		Client:       client,
		ManifestPath: "integration-tests/Cargo.toml",
	}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.NotNil(t, out)
	container.AssertCalled(t, "AsService", []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"})
	container.AssertCalled(t, "WithServiceBinding", "docker", dindService)
	container.AssertCalled(t, "WithEnvVariable", "DOCKER_HOST", "tcp://docker:2375")
	container.AssertCalled(t, "WithEnvVariable", "SHIPWRIGHT_DIND_INSTANCE", mock.Anything)
}
