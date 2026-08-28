// Package daggerkit provides adapters to convert real Dagger types to
// interfaces, mirroring internal/pipelines/dagger_adapter.go.
package daggerkit

import (
	"context"

	"dagger.io/dagger"
)

var _ DaggerClient = (*DaggerAdapter)(nil)

// DaggerAdapter adapts a real *dagger.Client to DaggerClient.
type DaggerAdapter struct {
	client *dagger.Client
}

// NewDaggerAdapter creates a new adapter for a real Dagger client.
func NewDaggerAdapter(client *dagger.Client) DaggerClient {
	return &DaggerAdapter{client: client}
}

// GetRealClient returns the underlying real Dagger client for use with shared functions.
func (a *DaggerAdapter) GetRealClient() *dagger.Client {
	return a.client
}

func (a *DaggerAdapter) Container() DaggerContainer {
	return &DaggerContainerAdapter{container: a.client.Container()}
}

func (a *DaggerAdapter) Host() DaggerHost {
	return &DaggerHostAdapter{host: a.client.Host()}
}

func (a *DaggerAdapter) CacheVolume(name string) DaggerCacheVolume {
	return &DaggerCacheVolumeAdapter{cache: a.client.CacheVolume(name)}
}

var _ DaggerHost = (*DaggerHostAdapter)(nil)

// DaggerHostAdapter adapts a real *dagger.Host to DaggerHost.
type DaggerHostAdapter struct {
	host *dagger.Host
}

func (a *DaggerHostAdapter) UnixSocket(path string) *dagger.Socket {
	return a.host.UnixSocket(path)
}

var _ DaggerCacheVolume = (*DaggerCacheVolumeAdapter)(nil)

// DaggerCacheVolumeAdapter adapts a real *dagger.CacheVolume to DaggerCacheVolume.
type DaggerCacheVolumeAdapter struct {
	cache *dagger.CacheVolume
}

var _ DaggerDirectory = (*DaggerDirectoryAdapter)(nil)

// DaggerDirectoryAdapter adapts a real *dagger.Directory to DaggerDirectory.
type DaggerDirectoryAdapter struct {
	directory *dagger.Directory
}

// NewDaggerDirectoryAdapter wraps a real *dagger.Directory as a
// DaggerDirectory, needed to feed a caller-supplied concrete Directory (e.g.
// a provider's public Build(source *dagger.Directory) parameter) into
// interface-typed methods such as DaggerContainer.WithMountedDirectory.
func NewDaggerDirectoryAdapter(directory *dagger.Directory) DaggerDirectory {
	return &DaggerDirectoryAdapter{directory: directory}
}

func (a *DaggerDirectoryAdapter) File(path string) DaggerFile {
	return &DaggerFileAdapter{file: a.directory.File(path)}
}

// GetRealDirectory returns the underlying real Dagger directory for use with shared functions.
func (a *DaggerDirectoryAdapter) GetRealDirectory() *dagger.Directory {
	return a.directory
}

var _ DaggerContainer = (*DaggerContainerAdapter)(nil)

// DaggerContainerAdapter adapts a real *dagger.Container to DaggerContainer.
type DaggerContainerAdapter struct {
	container *dagger.Container
}

func (a *DaggerContainerAdapter) From(image string) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.From(image)}
}

func (a *DaggerContainerAdapter) WithMountedDirectory(path string, dir DaggerDirectory) DaggerContainer {
	// A real adapter only ever receives another real adapter here; a mock
	// reaching this method body is a test wired to the wrong type.
	adapter, ok := dir.(*DaggerDirectoryAdapter)
	if !ok {
		panic("daggerkit: WithMountedDirectory called on DaggerContainerAdapter with a non-adapter DaggerDirectory")
	}
	return &DaggerContainerAdapter{container: a.container.WithMountedDirectory(path, adapter.directory)}
}

func (a *DaggerContainerAdapter) WithMountedCache(path string, cache DaggerCacheVolume) DaggerContainer {
	adapter, ok := cache.(*DaggerCacheVolumeAdapter)
	if !ok {
		panic("daggerkit: WithMountedCache called on DaggerContainerAdapter with a non-adapter DaggerCacheVolume")
	}
	return &DaggerContainerAdapter{container: a.container.WithMountedCache(path, adapter.cache)}
}

func (a *DaggerContainerAdapter) WithWorkdir(path string) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithWorkdir(path)}
}

func (a *DaggerContainerAdapter) WithEnvVariable(name, value string) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithEnvVariable(name, value)}
}

func (a *DaggerContainerAdapter) WithExec(args []string, opts DaggerContainerWithExecOpts) DaggerContainer {
	daggerOpts := dagger.ContainerWithExecOpts{
		RedirectStdout: opts.RedirectStdout,
	}
	return &DaggerContainerAdapter{container: a.container.WithExec(args, daggerOpts)}
}

func (a *DaggerContainerAdapter) WithNewFile(path, contents string) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithNewFile(path, contents)}
}

func (a *DaggerContainerAdapter) WithDirectory(path string, dir DaggerDirectory) DaggerContainer {
	adapter, ok := dir.(*DaggerDirectoryAdapter)
	if !ok {
		panic("daggerkit: WithDirectory called on DaggerContainerAdapter with a non-adapter DaggerDirectory")
	}
	return &DaggerContainerAdapter{container: a.container.WithDirectory(path, adapter.directory)}
}

func (a *DaggerContainerAdapter) WithEntrypoint(args []string) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithEntrypoint(args)}
}

func (a *DaggerContainerAdapter) WithRegistryAuth(address, username string, secret *dagger.Secret) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithRegistryAuth(address, username, secret)}
}

func (a *DaggerContainerAdapter) WithUnixSocket(path string, socket *dagger.Socket) DaggerContainer {
	return &DaggerContainerAdapter{container: a.container.WithUnixSocket(path, socket)}
}

func (a *DaggerContainerAdapter) File(path string) DaggerFile {
	return &DaggerFileAdapter{file: a.container.File(path)}
}

func (a *DaggerContainerAdapter) Directory(path string) DaggerDirectory {
	return &DaggerDirectoryAdapter{directory: a.container.Directory(path)}
}

func (a *DaggerContainerAdapter) Sync(ctx context.Context) (DaggerContainer, error) {
	synced, err := a.container.Sync(ctx)
	if err != nil {
		return nil, err
	}
	return &DaggerContainerAdapter{container: synced}, nil
}

func (a *DaggerContainerAdapter) Stdout(ctx context.Context) (string, error) {
	return a.container.Stdout(ctx)
}

func (a *DaggerContainerAdapter) Stderr(ctx context.Context) (string, error) {
	return a.container.Stderr(ctx)
}

func (a *DaggerContainerAdapter) Publish(ctx context.Context, address string) (string, error) {
	return a.container.Publish(ctx, address)
}

// GetRealContainer returns the underlying real Dagger container for use with shared functions.
func (a *DaggerContainerAdapter) GetRealContainer() *dagger.Container {
	return a.container
}

var _ DaggerFile = (*DaggerFileAdapter)(nil)

// DaggerFileAdapter adapts a real *dagger.File to DaggerFile.
type DaggerFileAdapter struct {
	file *dagger.File
}

func (a *DaggerFileAdapter) Contents(ctx context.Context) (string, error) {
	return a.file.Contents(ctx)
}
