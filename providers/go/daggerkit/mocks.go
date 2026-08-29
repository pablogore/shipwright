package daggerkit

import (
	"context"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
)

var _ DaggerClient = (*MockDaggerClient)(nil)

// MockDaggerClient implements DaggerClient using testify's mock package.
type MockDaggerClient struct {
	mock.Mock
}

func (m *MockDaggerClient) Container() DaggerContainer {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

var _ DaggerDirectory = (*MockDaggerDirectory)(nil)

// MockDaggerDirectory implements DaggerDirectory using testify's mock package.
type MockDaggerDirectory struct {
	mock.Mock
}

func (m *MockDaggerDirectory) GetRealDirectory() *dagger.Directory {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(*dagger.Directory)
	}
	return nil
}

func (m *MockDaggerDirectory) File(path string) DaggerFile {
	args := m.Called(path)
	if v := args.Get(0); v != nil {
		return v.(DaggerFile)
	}
	return nil
}

func (m *MockDaggerDirectory) Entries(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	var entries []string
	if v := args.Get(0); v != nil {
		entries = v.([]string)
	}
	return entries, args.Error(1)
}

var _ DaggerContainer = (*MockDaggerContainer)(nil)

// MockDaggerContainer implements DaggerContainer using testify's mock package.
type MockDaggerContainer struct {
	mock.Mock
}

func (m *MockDaggerContainer) From(image string) DaggerContainer {
	args := m.Called(image)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithMountedDirectory(path string, dir DaggerDirectory) DaggerContainer {
	args := m.Called(path, dir)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithWorkdir(path string) DaggerContainer {
	args := m.Called(path)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithEnvVariable(name, value string) DaggerContainer {
	args := m.Called(name, value)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithExec(execArgs []string, opts DaggerContainerWithExecOpts) DaggerContainer {
	args := m.Called(execArgs, opts)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithNewFile(path, contents string) DaggerContainer {
	args := m.Called(path, contents)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithDirectory(path string, dir DaggerDirectory) DaggerContainer {
	args := m.Called(path, dir)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithEntrypoint(execArgs []string) DaggerContainer {
	args := m.Called(execArgs)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) WithRegistryAuth(address, username string, secret *dagger.Secret) DaggerContainer {
	args := m.Called(address, username, secret)
	if v := args.Get(0); v != nil {
		return v.(DaggerContainer)
	}
	return nil
}

func (m *MockDaggerContainer) File(path string) DaggerFile {
	args := m.Called(path)
	if v := args.Get(0); v != nil {
		return v.(DaggerFile)
	}
	return nil
}

func (m *MockDaggerContainer) Directory(path string) DaggerDirectory {
	args := m.Called(path)
	if v := args.Get(0); v != nil {
		return v.(DaggerDirectory)
	}
	return nil
}

func (m *MockDaggerContainer) Sync(ctx context.Context) (DaggerContainer, error) {
	args := m.Called(ctx)
	var c DaggerContainer
	if v := args.Get(0); v != nil {
		c = v.(DaggerContainer)
	}
	return c, args.Error(1)
}

func (m *MockDaggerContainer) Stdout(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockDaggerContainer) Publish(ctx context.Context, address string) (string, error) {
	args := m.Called(ctx, address)
	return args.String(0), args.Error(1)
}

func (m *MockDaggerContainer) GetRealContainer() *dagger.Container {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(*dagger.Container)
	}
	return nil
}

var _ DaggerFile = (*MockDaggerFile)(nil)

// MockDaggerFile implements DaggerFile using testify's mock package.
type MockDaggerFile struct {
	mock.Mock
}

func (m *MockDaggerFile) GetRealFile() *dagger.File {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(*dagger.File)
	}
	return nil
}

func (m *MockDaggerFile) Contents(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}
