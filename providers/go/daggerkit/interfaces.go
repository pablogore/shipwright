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
// mocking. File and Entries were added for GoRuntimeInspector (design.md
// D-9): reading a workspace's go.work/go.mod/.go-version content is the
// first read-side consumer this module has ever had. WithNewFile is the
// write side (design.md D-9), added for GoRuntimeUpgrader: since
// dagger.Directory is an immutable value, WithNewFile returns a new
// DaggerDirectory rather than mutating in place, which is what gives the
// no-partial-mutation guarantee (analysis completes before the first
// WithNewFile call; a failed run never returns a directory).
type DaggerDirectory interface {
	// GetRealDirectory returns the underlying real Dagger directory (only for adapters)
	GetRealDirectory() *dagger.Directory
	// File returns a handle to a file at path within this directory,
	// without reading it — reading happens via the returned DaggerFile's
	// Contents.
	File(path string) DaggerFile
	// Entries lists the names of files and directories at this
	// directory's root, used to discover which tier-1 sources
	// (go.work, .go-version, go.mod) are actually present.
	Entries(ctx context.Context) ([]string, error)
	// WithNewFile returns a new DaggerDirectory with the file at path
	// replaced by contents (or created, if absent). The receiver is left
	// unmodified.
	WithNewFile(path, contents string) DaggerDirectory
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
// Contents(context.Context) was added for GoRuntimeInspector (design.md
// D-9): it is the first consumer in this module that reads a file's
// content in production code, rather than only returning the File to its
// caller unread. This package also keeps GetRealFile, absent from the root
// package, because every one of GoLinter / GoUnitTester / GoVulnScanner's
// Test methods returns a concrete *dagger.File and therefore must unwrap
// one.
type DaggerFile interface {
	// GetRealFile returns the underlying real Dagger file (only for adapters)
	GetRealFile() *dagger.File
	// Contents reads this file's full content.
	Contents(ctx context.Context) (string, error)
}
