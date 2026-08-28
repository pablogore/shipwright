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

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

func TestContainerPublisher_Publish_NilClient(t *testing.T) {
	publisher := &golang.ContainerPublisher{}

	ref, err := publisher.Publish(context.Background(), nil, "ghcr.io/acme/api:v1", nil)

	if err == nil {
		t.Fatal("ContainerPublisher.Publish() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("ContainerPublisher.Publish() error = %v, want it to mention an unconfigured client", err)
	}
	if ref != "" {
		t.Fatalf("ContainerPublisher.Publish() ref = %q, want empty on error", ref)
	}
}

// TestContainerPublisher_Publish_Success proves the happy path: the build
// Directory is staged into the base image, the entrypoint binary is
// chmod'd and set, registry auth is attached from the configured
// username plus the caller's Secret, and the resolved published ref is
// returned unwrapped.
func TestContainerPublisher_Publish_Success(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	build := &dagger.Directory{}
	secret := &dagger.Secret{}
	const publishedRef = "ghcr.io/acme/api@sha256:abc123"

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "alpine:latest").Return(mockContainer)
	mockContainer.On("WithDirectory", "/app", mock.MatchedBy(func(d daggerkit.DaggerDirectory) bool {
		return d.GetRealDirectory() == build
	})).Return(mockContainer)
	mockContainer.On("WithExec", []string{"test", "-f", "/app/app"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(mockContainer, nil)
	mockContainer.On("WithExec", []string{"chmod", "+x", "/app/app"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("WithEntrypoint", []string{"/app/app"}).Return(mockContainer)
	mockContainer.On("WithRegistryAuth", "ghcr.io", "acme-user", secret).Return(mockContainer)
	mockContainer.On("Publish", mock.Anything, "ghcr.io/acme/api:v1").Return(publishedRef, nil)

	publisher := &golang.ContainerPublisher{
		Client: mockClient,
		Config: shipwright.ArtifactConfig{RegistryUser: "acme-user"},
	}

	ref, err := publisher.Publish(context.Background(), build, "ghcr.io/acme/api:v1", secret)

	require.NoError(t, err)
	assert.Equal(t, publishedRef, ref)
	mockContainer.AssertExpectations(t)
}

// TestContainerPublisher_Publish_BinaryNameMismatch is the mock-based
// counterpart of TestContainerPublisher_Publish_BinaryNameMismatch_RealEngine
// (integration_test.go, PR #176 review finding #7): when the expected
// entrypoint binary is missing from the staged image, Publish must fail
// with an actionable message naming the missing path, and must never
// reach the chmod/entrypoint/publish steps.
func TestContainerPublisher_Publish_BinaryNameMismatch(t *testing.T) {
	mockClient := &daggerkit.MockDaggerClient{}
	mockContainer := &daggerkit.MockDaggerContainer{}
	build := &dagger.Directory{}
	checkErr := errors.New("exit code 1: test: /app/app: no such file or directory")

	mockClient.On("Container").Return(mockContainer)
	mockContainer.On("From", "alpine:latest").Return(mockContainer)
	mockContainer.On("WithDirectory", "/app", mock.Anything).Return(mockContainer)
	mockContainer.On("WithExec", []string{"test", "-f", "/app/app"}, daggerkit.DaggerContainerWithExecOpts{}).Return(mockContainer)
	mockContainer.On("Sync", mock.Anything).Return(nil, checkErr)

	publisher := &golang.ContainerPublisher{Client: mockClient}

	ref, err := publisher.Publish(context.Background(), build, "ghcr.io/acme/api:v1", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected binary at \"/app/app\" not found in container")
	assert.Empty(t, ref)
	mockContainer.AssertNotCalled(t, "WithEntrypoint", mock.Anything)
	mockContainer.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}
