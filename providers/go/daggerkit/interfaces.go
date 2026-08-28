// Package daggerkit provides interfaces for Dagger types to enable mocking
// in this module's provider unit tests, mirroring the root module's
// internal/daggerkit package. This is a standalone copy, not an import of
// the root package: providers/go is its own Go module and must not depend
// on the root module (naming_test.go / internalimport_test.go enforce this
// module's independence from anything under the root's internal/**).
//
// Scoped down from the root package's full surface to only the methods
// this module's five capability providers (GoBuilder, GoLinter,
// GoUnitTester, GoVulnScanner, ContainerPublisher) actually call: no
// DaggerHost, DaggerCacheVolume, or DaggerClient.CacheVolume/Host, since
// none of them mount a cache volume or touch the host filesystem.
package daggerkit

import (
	"context"

	"dagger.io/dagger"
)

// DaggerClient interface abstracts the dagger.Client to enable mocking.
type DaggerClient interface {
	Container() DaggerContainer
}

// DaggerDirectory interface abstracts the dagger.Directory to enable
// mocking. No File(string) method: unlike the root package, nothing here
// ever calls a method on a DaggerDirectory produced by another
// DaggerDirectory — it is only ever constructed from a caller-supplied
// concrete *dagger.Directory and passed opaquely into WithMountedDirectory
// / WithDirectory.
type DaggerDirectory interface {
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
	WithWorkdir(string) DaggerContainer
	WithEnvVariable(string, string) DaggerContainer
	WithExec([]string, DaggerContainerWithExecOpts) DaggerContainer
	WithNewFile(string, string) DaggerContainer
	WithDirectory(string, DaggerDirectory) DaggerContainer
	WithEntrypoint([]string) DaggerContainer
	// WithRegistryAuth takes a concrete *dagger.Secret directly rather than
	// a wrapping interface: ContainerPublisher only ever receives it
	// opaquely through its own Publish(..., creds *dagger.Secret)
	// parameter and forwards it untouched, never calling a method on it.
	WithRegistryAuth(string, string, *dagger.Secret) DaggerContainer
	File(string) DaggerFile
	Directory(string) DaggerDirectory
	Sync(context.Context) (DaggerContainer, error)
	Stdout(context.Context) (string, error)
	Publish(context.Context, string) (string, error)
	// GetRealContainer returns the underlying real Dagger container (only for adapters)
	GetRealContainer() *dagger.Container
}

// DaggerFile interface abstracts the dagger.File to enable mocking.
//
// No Contents(context.Context) method: unlike the root package, nothing
// here ever reads a report File's contents in production code — every
// capability's public method returns the File to its caller unread. This
// package instead adds GetRealFile, absent from the root package, because
// every one of GoLinter / GoUnitTester / GoVulnScanner's Test methods
// returns a concrete *dagger.File and therefore must unwrap one.
type DaggerFile interface {
	// GetRealFile returns the underlying real Dagger file (only for adapters)
	GetRealFile() *dagger.File
}
