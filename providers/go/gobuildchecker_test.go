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

func TestGoBuildChecker_Test_NilClient(t *testing.T) {
	checker := &golang.GoBuildChecker{}

	out, err := checker.Test(context.Background(), &dagger.Directory{})

	if err == nil {
		t.Fatal("GoBuildChecker.Test() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoBuildChecker.Test() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoBuildChecker.Test() file = %v, want nil on error", out)
	}
}

func TestGoBuildChecker_Test_NilSource(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	checker := &golang.GoBuildChecker{Client: mockClient}

	out, err := checker.Test(context.Background(), nil)

	if err == nil {
		t.Fatal("GoBuildChecker.Test() error = nil, want error for nil source directory")
	}
	if !strings.Contains(err.Error(), "source directory is nil") {
		t.Fatalf("GoBuildChecker.Test() error = %v, want it to mention a nil source directory", err)
	}
	if out != nil {
		t.Fatalf("GoBuildChecker.Test() file = %v, want nil on error", out)
	}
}

// TestGoBuildChecker_Test_Success proves the clean-build path: since `go
// build ./...` prints nothing on success, the report body is synthesized
// (design.md D-5) rather than passed through verbatim, and no `go mod tidy`
// is ever invoked (design.md D-3).
func TestGoBuildChecker_Test_Success(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockReportFile := &daggerkit.MockDaggerFile{}
	src := &dagger.Directory{}
	realReportFile := &dagger.File{}
	const buildOutput = ""

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.26.7").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockClient.On("CacheVolume", "shipwright-go-mod-cache").Return(&daggerkit.MockDaggerCacheVolume{})
	mockClient.On("CacheVolume", "shipwright-go-build-cache").Return(&daggerkit.MockDaggerCacheVolume{})
	mockContainer.On("WithMountedCache", "/go/pkg/mod", mock.Anything).Return(mockContainer)
	mockContainer.On("WithMountedCache", "/root/.cache/go-build", mock.Anything).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return(buildOutput, nil)
	mockContainer.On("WithNewFile", "/tmp/build-check-report.txt", mock.MatchedBy(func(body string) bool {
		return strings.Contains(body, "go build ./... succeeded") && strings.Contains(body, "golang:1.26.7")
	})).Return(mockContainer)
	mockContainer.On("File", "/tmp/build-check-report.txt").Return(mockReportFile)
	mockReportFile.On("GetRealFile").Return(realReportFile)

	checker := &golang.GoBuildChecker{Client: mockClient}

	out, err := checker.Test(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realReportFile, out)
	mockContainer.AssertExpectations(t)
	mockContainer.AssertNotCalled(t, "WithExec", []string{"go", "mod", "tidy"}, mock.Anything)
}

// TestGoBuildChecker_Test_GoVersionOverride pins design.md D-6: unlike
// GoVulnScanner/GoLinter's GoVersion field (never wired into register.go's
// WithSchema), GoBuildChecker's GoVersion actually selects the toolchain
// image, mirroring GoBuilder's own resolveGoVersion usage.
func TestGoBuildChecker_Test_GoVersionOverride(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockReportFile := &daggerkit.MockDaggerFile{}
	src := &dagger.Directory{}
	realReportFile := &dagger.File{}
	const buildOutput = ""

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.25.5").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockClient.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})
	mockContainer.On("WithMountedCache", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return(buildOutput, nil)
	mockContainer.On("WithNewFile", "/tmp/build-check-report.txt", mock.Anything).Return(mockContainer)
	mockContainer.On("File", "/tmp/build-check-report.txt").Return(mockReportFile)
	mockReportFile.On("GetRealFile").Return(realReportFile)

	checker := &golang.GoBuildChecker{Client: mockClient, GoVersion: "1.25.5"}

	out, err := checker.Test(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realReportFile, out)
	mockContainer.AssertExpectations(t)
}

// TestGoBuildChecker_Test_CompileFailure_Fails proves that a non-zero `go
// build ./...` exit surfaces as a wrapped "gobuildchecker:" error carrying
// the compiler's diagnostic text (which Dagger's ExecError embeds in
// err.Error(), design.md D-4), and that no report file is ever written for
// a failed build.
func TestGoBuildChecker_Test_CompileFailure_Fails(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	src := &dagger.Directory{}
	buildErr := errors.New("exit code 1: ./pkg/foo/foo.go:12:2: undefined: bar")

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.26.7").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GO111MODULE", "on").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockClient.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})
	mockContainer.On("WithMountedCache", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Stdout", mock.Anything).Return("", buildErr)

	checker := &golang.GoBuildChecker{Client: mockClient}

	out, err := checker.Test(context.Background(), src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gobuildchecker: go build ./... failed")
	assert.Contains(t, err.Error(), "undefined: bar")
	assert.Nil(t, out)
	mockContainer.AssertNotCalled(t, "WithNewFile", mock.Anything, mock.Anything)
}
