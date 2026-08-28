package rust_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

func TestRustUnitTester_Test_NilClient(t *testing.T) {
	tester := &rust.RustUnitTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("RustUnitTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustUnitTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustUnitTester.Test() file = %v, want nil on error", out)
	}
}

// mockUnitTestContainer builds the mocked container chain shared by both
// coverage-threshold mock tests below, parameterized by the
// cargo-tarpaulin Stdout output the coverage step should observe.
func mockUnitTestContainer(t *testing.T, tarpaulinStdout string) (*daggerkit.MockDaggerClient, *daggerkit.MockDaggerContainer) {
	t.Helper()

	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithMountedCache", mock.Anything, mock.Anything).Return(container)
	container.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithWorkdir", mock.Anything).Return(container)
	container.On("WithExec", []string{"cargo", "test", "--workspace"}).Return(container)
	container.On("Stdout", mock.Anything).Return("running 1 test ... test result: ok\n", nil)

	tarpaulinContainer := &daggerkit.MockDaggerContainer{}
	container.On("WithExec", []string{"cargo", "install", "cargo-tarpaulin", "--locked", "--version", "0.37.2"}).Return(tarpaulinContainer)
	tarpaulinContainer.On("WithExec", []string{"cargo", "tarpaulin", "--out", "Stdout", "--engine", "llvm", "--workspace"}).Return(tarpaulinContainer)
	tarpaulinContainer.On("Stdout", mock.Anything).Return(tarpaulinStdout, nil)

	realFile := &dagger.File{}
	reportFile := &daggerkit.MockDaggerFile{}
	reportFile.On("GetRealFile").Return(realFile)
	container.On("WithNewFile", mock.Anything, mock.Anything).Return(container)
	container.On("File", mock.Anything).Return(reportFile)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	return client, container
}

// TestRustUnitTester_Test_MockClient_PassesWithinThreshold mirrors
// TestRustUnitTester_Test_RealEngine_PassesWithinThreshold: a coverage
// percentage at or above Config.Coverage must let Test succeed.
func TestRustUnitTester_Test_MockClient_PassesWithinThreshold(t *testing.T) {
	client, _ := mockUnitTestContainer(t, "87.50% coverage, 10/12 lines covered\n")

	tester := &rust.RustUnitTester{
		Client: client,
		Config: shipwright.TestConfig{Coverage: 50},
	}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.NoError(t, err)
	require.NotNil(t, out)
}

// TestRustUnitTester_Test_MockClient_FailsBelowThreshold covers the inverse
// branch RustUnitTester.enforceCoverageThreshold guards: a coverage
// percentage under Config.Coverage must fail Test with an error naming the
// shortfall, with no real engine involved.
func TestRustUnitTester_Test_MockClient_FailsBelowThreshold(t *testing.T) {
	client, _ := mockUnitTestContainer(t, "10.00% coverage, 1/10 lines covered\n")

	tester := &rust.RustUnitTester{
		Client: client,
		Config: shipwright.TestConfig{Coverage: 50},
	}

	out, err := tester.Test(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "below the required threshold")
	require.Nil(t, out)
}
