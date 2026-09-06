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

func TestContainerPublisher_Publish_NilClient(t *testing.T) {
	publisher := &rust.ContainerPublisher{}

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

// TestContainerPublisher_Publish_MockClient_BinaryNameMismatch mirrors
// TestContainerPublisher_Publish_BinaryNameMismatch_RealEngine (PR #176
// review finding #7): when Config.BinaryName is left at its "app" default
// but the staged container doesn't have a binary at that path, Publish must
// fail with an actionable message naming the missing entrypoint, instead of
// chmod's opaque "no such file or directory" -- proven here via a Sync
// failure on the "test -f <entrypoint>" step, with no real engine involved.
func TestContainerPublisher_Publish_MockClient_BinaryNameMismatch(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := &daggerkit.MockDaggerContainer{}
	container.On("From", mock.Anything).Return(container)
	container.On("WithDirectory", mock.Anything, mock.Anything).Return(container)
	container.On("WithExec", []string{"test", "-f", "/app/app"}).Return(container)
	container.On("Sync", mock.Anything).Return(nil, &dagger.ExecError{
		ExitCode: 1,
		Stderr:   "test: /app/app: no such file or directory\n",
	})

	client.On("Container").Return(container)

	publisher := &rust.ContainerPublisher{Client: client}

	ref, err := publisher.Publish(context.Background(), &dagger.Directory{}, "ghcr.io/acme/api:v1", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "expected binary at")
	require.Contains(t, err.Error(), "not found in container")
	require.Empty(t, ref)
	container.AssertExpectations(t)
	container.AssertNotCalled(t, "WithEntrypoint", mock.Anything)
}
