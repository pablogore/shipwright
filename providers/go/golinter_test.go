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

func TestGoLinter_Test_NilClient(t *testing.T) {
	linter := &golang.GoLinter{}

	out, err := linter.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoLinter.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoLinter.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoLinter.Test() file = %v, want nil on error", out)
	}
}

// TestGoLinter_Test_Success proves the clean-lint path: golangci-lint's
// captured stdout is written to a new in-container file and returned as
// the report File, via mocks -- GoLinter has no _RealEngine test of its
// own, so this is this capability's first coverage of the success path.
func TestGoLinter_Test_Success(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockReportFile := &daggerkit.MockDaggerFile{}
	src := &dagger.Directory{}
	realReportFile := &dagger.File{}
	const lintOutput = "0 issues.\n"

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golangci/golangci-lint:v2.13.2").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", []string{"golangci-lint", "run", "--timeout", "5m", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return(lintOutput, nil)
	mockContainer.On("WithNewFile", "/tmp/lint-report.txt", lintOutput).Return(mockContainer)
	mockContainer.On("File", "/tmp/lint-report.txt").Return(mockReportFile)
	mockReportFile.On("GetRealFile").Return(realReportFile)

	linter := &golang.GoLinter{Client: mockClient}

	out, err := linter.Test(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realReportFile, out)
	mockContainer.AssertExpectations(t)
}

// TestGoLinter_Test_LintFindings_Fails proves that a non-zero
// golangci-lint exit (reported as an error from Stdout) surfaces as a
// wrapped "golinter:" error and a nil report File, instead of writing a
// report from a failed run.
func TestGoLinter_Test_LintFindings_Fails(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	src := &dagger.Directory{}
	lintErr := errors.New("exit code 1: main.go:10:2: unused variable x (unused)")

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golangci/golangci-lint:v2.13.2").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return("", lintErr)

	linter := &golang.GoLinter{Client: mockClient}

	out, err := linter.Test(context.Background(), src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "golinter: golangci-lint found issues")
	assert.Nil(t, out)
	mockContainer.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}
