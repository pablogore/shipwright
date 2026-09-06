// Package daggerkit provides interfaces for Dagger types to enable mocking
// in provider unit tests, mirroring internal/pipelines/dagger_interfaces.go.
package daggerkit

import (
	"context"

	"dagger.io/dagger"
)

// DaggerClient interface abstracts the dagger.Client to enable mocking.
type DaggerClient interface {
	Container() DaggerContainer
	Host() DaggerHost
	CacheVolume(string) DaggerCacheVolume
	// GetRealClient returns the underlying real Dagger client (only for adapters)
	GetRealClient() *dagger.Client
}

// DaggerHost interface abstracts the dagger.Host to enable mocking.
type DaggerHost interface {
	// UnixSocket returns *dagger.Socket directly rather than a wrapping
	// interface: callers only ever pass it through opaquely to
	// DaggerContainer.WithUnixSocket, never call a method on it.
	UnixSocket(string) *dagger.Socket
}

// DaggerCacheVolume interface abstracts the dagger.CacheVolume to enable
// mocking.
type DaggerCacheVolume interface {
	// Add methods that are actually used by providers. None are today: a
	// CacheVolume is only ever passed opaquely into WithMountedCache.
}

// DaggerDirectory interface abstracts the dagger.Directory to enable
// mocking.
type DaggerDirectory interface {
	File(string) DaggerFile
	// GetRealDirectory returns the underlying real Dagger directory (only for adapters)
	GetRealDirectory() *dagger.Directory
}

// DaggerContainerWithExecOpts represents options for container exec
// operations.
type DaggerContainerWithExecOpts struct {
	RedirectStdout string
}

// DaggerContainer interface abstracts the dagger.Container to enable
// mocking.
type DaggerContainer interface {
	From(string) DaggerContainer
	WithMountedDirectory(string, DaggerDirectory) DaggerContainer
	WithMountedCache(string, DaggerCacheVolume) DaggerContainer
	WithWorkdir(string) DaggerContainer
	WithEnvVariable(string, string) DaggerContainer
	WithExec([]string, DaggerContainerWithExecOpts) DaggerContainer
	WithNewFile(string, string) DaggerContainer
	WithDirectory(string, DaggerDirectory) DaggerContainer
	WithEntrypoint([]string) DaggerContainer
	// WithRegistryAuth and WithUnixSocket take *dagger.Secret/*dagger.Socket
	// directly rather than a wrapping interface, same rationale as
	// DaggerHost.UnixSocket above.
	WithRegistryAuth(string, string, *dagger.Secret) DaggerContainer
	WithUnixSocket(string, *dagger.Socket) DaggerContainer
	File(string) DaggerFile
	Directory(string) DaggerDirectory
	Sync(context.Context) (DaggerContainer, error)
	Stdout(context.Context) (string, error)
	Stderr(context.Context) (string, error)
	Publish(context.Context, string) (string, error)
	// GetRealContainer returns the underlying real Dagger container (only for adapters)
	GetRealContainer() *dagger.Container
}

// DaggerFile interface abstracts the dagger.File to enable mocking.
type DaggerFile interface {
	Contents(context.Context) (string, error)
}
