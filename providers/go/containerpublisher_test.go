package golang_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pablogore/shipwright/providers/go"
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
