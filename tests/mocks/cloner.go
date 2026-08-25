// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"dagger.io/dagger"
	"github.com/pablogore/shipwright/internal/pipelines/shared"
)

// MockCloner es un mock del clonador para tests
type MockCloner struct {
	MockClone func(ctx context.Context, client *dagger.Client, opts shared.GitCloneOpts) (*dagger.Directory, error)
}

// Clone implementa la interfaz Cloner
func (m *MockCloner) Clone(ctx context.Context, client *dagger.Client, opts shared.GitCloneOpts) (*dagger.Directory, error) {
	if m.MockClone != nil {
		return m.MockClone(ctx, client, opts)
	}
	// By default, return an empty directory
	return client.Directory(), nil
}
