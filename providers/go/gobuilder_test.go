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

func TestGoBuilder_Build_NilClient(t *testing.T) {
	builder := &golang.GoBuilder{}

	out, err := builder.Build(context.Background(), nil)

	if err == nil {
		t.Fatal("GoBuilder.Build() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("GoBuilder.Build() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("GoBuilder.Build() directory = %v, want nil on error", out)
	}
}

// TestGoBuilder_Build_Success is the mock-based counterpart of
// TestGoBuilder_Build_RealEngine (integration_test.go): it proves the same
// build-success business logic -- default Go version/binary name
// resolution, the mount/build/sync call chain, and unwrapping the
// resulting Directory -- without a real Dagger engine.
func TestGoBuilder_Build_Success(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	mockOutputDir := &daggerkit.MockDaggerDirectory{}
	src := &dagger.Directory{}
	realOutputDir := &dagger.Directory{}

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.26.7").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.MatchedBy(func(d daggerkit.DaggerDirectory) bool {
		return d.GetRealDirectory() == src
	})).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GOPATH", "/go").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "mod", "tidy"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("WithExec", []string{"go", "build", "-ldflags=-s -w", "-o", "/output/app", "."}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("Directory", "/output").Return(mockOutputDir)
	mockOutputDir.On("GetRealDirectory").Return(realOutputDir)

	builder := &golang.GoBuilder{Client: mockClient}

	out, err := builder.Build(context.Background(), src)

	require.NoError(t, err)
	assert.Same(t, realOutputDir, out)
	mockClient.AssertExpectations(t)
	mockContainer.AssertExpectations(t)
	mockOutputDir.AssertExpectations(t)
}

// TestGoBuilder_Build_Failure_Stderr proves the failed-build path: when
// the container's Sync fails (e.g. a compile error), Build wraps the
// error -- which carries the failing command's stderr, as Dagger's own
// ExecError does -- with a "gobuilder:" prefix, and returns a nil
// Directory. No real engine is needed to prove this: the wrap-and-nil-out
// behavior lives entirely in GoBuilder.Build, not in Dagger itself.
func TestGoBuilder_Build_Failure_Stderr(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	src := &dagger.Directory{}
	execErr := errors.New("process \"go build\" did not complete successfully: exit code 2: stderr: ./main.go:3:2: undefined: foo")

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "golang:1.26.7").Return(mockContainer)
	mockContainer.On("WithMountedDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithWorkdir", "/app").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "GOPATH", "/go").Return(mockContainer)
	mockContainer.On("WithEnvVariable", "CGO_ENABLED", "0").Return(mockContainer)
	mockContainer.On("WithExec", mock.Anything, mock.Anything).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, execErr)

	builder := &golang.GoBuilder{Client: mockClient}

	out, err := builder.Build(context.Background(), src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gobuilder: failed to build go binary")
	assert.Contains(t, err.Error(), "stderr: ./main.go:3:2: undefined: foo")
	assert.Nil(t, out)
	mockContainer.AssertNotCalled(t, "Directory", mock.Anything)
}
