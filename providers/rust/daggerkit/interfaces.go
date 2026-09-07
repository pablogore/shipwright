// Package daggerkit provides interfaces for Dagger types to enable mocking
// in provider unit tests. Scoped-down copy of the root module's
// internal/daggerkit, kept local to providers/rust (its own Go module) so
// this module gains no dependency on the root module — see
// internalimport_test.go's D5 guard, which is extended (not disabled) to
// permit this module's own internal/daggerkit while still forbidding a
// reach into the root's internal/**.
//
// Deliberately narrower than the root's surface: only the DaggerClient
// methods and chained Container/Directory/File/Host/CacheVolume methods
// providers/rust's capability implementations actually call are present.
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
}

// DaggerHost interface abstracts the dagger.Host to enable mocking.
type DaggerHost interface {
	// UnixSocket returns *dagger.Socket directly rather than a wrapping
	// interface: callers only ever pass it through opaquely to
	// DaggerContainer.WithUnixSocket, never call a method on it.
	UnixSocket(string) *dagger.Socket
}

// DaggerCacheVolume interface abstracts the dagger.CacheVolume to enable
// mocking. No methods: every provider in this module only ever passes a
// CacheVolume opaquely into WithMountedCache.
type DaggerCacheVolume interface {
}

// DaggerService interface abstracts the dagger.Service to enable mocking. No
// methods: every provider in this module only ever passes a Service opaquely
// into WithServiceBinding (e.g. a Docker-in-Docker daemon container turned
// into a service via AsService).
type DaggerService interface {
}

// DaggerDirectory interface abstracts the dagger.Directory to enable
// mocking.
type DaggerDirectory interface {
	File(string) DaggerFile
	// GetRealDirectory returns the underlying real Dagger directory (only
	// for adapters) — needed to unwrap RustBuilder.Build's returned
	// Directory back to the concrete *dagger.Directory its public signature
	// promises.
	GetRealDirectory() *dagger.Directory
}

// DaggerContainer interface abstracts the dagger.Container to enable
// mocking.
type DaggerContainer interface {
	From(string) DaggerContainer
	WithMountedDirectory(string, DaggerDirectory) DaggerContainer
	WithMountedCache(string, DaggerCacheVolume) DaggerContainer
	WithWorkdir(string) DaggerContainer
	WithExec([]string) DaggerContainer
	WithNewFile(string, string) DaggerContainer
	WithDirectory(string, DaggerDirectory) DaggerContainer
	WithEntrypoint([]string) DaggerContainer
	// WithRegistryAuth and WithUnixSocket take *dagger.Secret/*dagger.Socket
	// directly rather than a wrapping interface, same rationale as
	// DaggerHost.UnixSocket above.
	WithRegistryAuth(string, string, *dagger.Secret) DaggerContainer
	WithUnixSocket(string, *dagger.Socket) DaggerContainer
	WithEnvVariable(string, string) DaggerContainer
	// WithExposedPort, AsService and WithServiceBinding exist for the
	// Docker-in-Docker sidecar pattern (see providers/rust/dockerdaemon.go):
	// a privileged dockerd container is exposed as a DaggerService and bound
	// into the caller's container under a network alias, giving
	// testcontainers-rs a real, routable Docker daemon instead of a mounted
	// host socket (which Dagger's own container sandboxing model makes
	// unreachable for sibling-container networking — see dockerdaemon.go's
	// doc comment for the full root cause).
	WithExposedPort(int) DaggerContainer
	// AsService runs args as the service's live process, with Dagger's
	// InsecureRootCapabilities (needed to start dockerd itself). Args must
	// be passed here rather than via a prior WithExec: WithExec's semantics
	// are "run to completion and snapshot the result," which a daemon that
	// runs forever never does — see dockerdaemon.go's dockerdCommand comment.
	AsService([]string) DaggerService
	WithServiceBinding(string, DaggerService) DaggerContainer
	File(string) DaggerFile
	Directory(string) DaggerDirectory
	Sync(context.Context) (DaggerContainer, error)
	Stdout(context.Context) (string, error)
	Stderr(context.Context) (string, error)
	Publish(context.Context, string) (string, error)
}

// DaggerFile interface abstracts the dagger.File to enable mocking.
type DaggerFile interface {
	Contents(context.Context) (string, error)
	// GetRealFile returns the underlying real Dagger file (only for
	// adapters) — needed by every Tester in this module, each of which
	// returns its report as a *dagger.File.
	GetRealFile() *dagger.File
}
