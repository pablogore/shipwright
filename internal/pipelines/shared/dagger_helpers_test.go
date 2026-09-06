package shared

import (
	"context"
	"testing"

	"dagger.io/dagger"
)

// requireDaggerClient connects to a Dagger engine for a test that genuinely
// needs one, skipping (not failing) the test when no engine is reachable.
// Unit/static tests in this package must never depend on this: only call it
// from tests that actually exercise a *dagger.Client or *dagger.Container.
func requireDaggerClient(ctx context.Context, t *testing.T) *dagger.Client {
	t.Helper()

	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Skipf("skipping: no Dagger/Docker engine available: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return client
}
