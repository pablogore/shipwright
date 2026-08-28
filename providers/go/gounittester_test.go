package golang_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestGoUnitTester_Test_NilClient(t *testing.T) {
	tester := &golang.GoUnitTester{}

	out, err := tester.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoUnitTester.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoUnitTester.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoUnitTester.Test() file = %v, want nil on error", out)
	}
}

// TestGoUnitTester_Test_PassesWithinThreshold is the mock-based
// counterpart of TestGoUnitTester_Test_RealEngine_PassesWithinThreshold
// (integration_test.go): it proves the same threshold-pass business logic
// -- coverage parsed from `go tool cover -func` output, compared against
// Config.Coverage, and the coverage profile File returned unwrapped -- but
// this one asserts the container was built with CGO_ENABLED=1 (the
// regression this package's real-engine test itself guards), which the
// mock lets fail loudly instead of only under a real `-race` compile.
func TestGoUnitTester_Test_PassesWithinThreshold(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockCoverageFile := &daggerkit.MockDaggerFile{}
	src := &dagger.Directory{}
	realCoverageFile := &dagger.File{}

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.25.5").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/src", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/src").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "1").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "test", "-v", "-race", "-coverprofile=/tmp/coverage.out", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("WithExec", []string{"go", "tool", "cover", "-func=/tmp/coverage.out"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return("total:\t\t\t\t(statements)\t87.50%\n", nil)
	mockContainer.On("File", "/tmp/coverage.out").Return(mockCoverageFile)
	mockCoverageFile.On("GetRealFile").Return(realCoverageFile)

	tester := &golang.GoUnitTester{Client: mockClient, Config: shipwright.TestConfig{Coverage: 1}}

	out, err := tester.Test(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realCoverageFile, out)
	mockContainer.AssertExpectations(t)
}

// TestGoUnitTester_Test_BelowThreshold_Fails proves the threshold-fail
// business logic that has no real-engine counterpart at all: coverage
// below Config.Coverage must fail the run with a message naming both
// percentages, and -- since the threshold check runs before the coverage
// profile File is read back -- the run must never call Container.File in
// that case.
func TestGoUnitTester_Test_BelowThreshold_Fails(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	src := &dagger.Directory{}

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.25.5").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/src", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/src").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "1").Return(mockContainer)
	mockContainer.On("WithExec", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("Stdout", mock.Anything).Return("total:\t\t\t\t(statements)\t50.00%\n", nil)

	tester := &golang.GoUnitTester{Client: mockClient, Config: shipwright.TestConfig{Coverage: 90}}

	out, err := tester.Test(context.Background(), src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "coverage 50.00% is below the required threshold of 90.00%")
	assert.Nil(t, out)
	mockContainer.AssertNotCalled(t, "File", mock.Anything)
}
