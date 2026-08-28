package rust_test

import (
	"context"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

func TestRustBuilder_Build_NilClient(t *testing.T) {
	builder := &rust.RustBuilder{}

	out, err := builder.Build(context.Background(), nil)

	if err == nil {
		t.Fatal("RustBuilder.Build() error = nil, want error for unconfigured client")
	}
	if !strings.Contains(err.Error(), "dagger client is not configured") {
		t.Fatalf("RustBuilder.Build() error = %v, want it to mention an unconfigured client", err)
	}
	if out != nil {
		t.Fatalf("RustBuilder.Build() directory = %v, want nil on error", out)
	}
}

// TestRustBuilder_Build_NilSource_MockClient mirrors
// TestRustBuilder_Build_NilSource_RealClient (integration_test.go) via a
// mock client instead of a real engine connection: the nil-source guard
// returns before any Client method is called, so no expectation needs
// setting on the mock at all.
func TestRustBuilder_Build_NilSource_MockClient(t *testing.T) {
	builder := &rust.RustBuilder{Client: &daggerkit.MockDaggerClient{}}

	_, err := builder.Build(context.Background(), nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "source directory is nil")
}

// mockBuildContainer wires a MockDaggerContainer to accept RustBuilder's
// entire chained container-building sequence, returning itself from every
// chained call so the fluent chain in rustbuilder.go's Build resolves to one
// object -- mirroring how a real *dagger.Container chain always returns
// itself (a new immutable value, but the same conceptual pipeline).
func mockBuildContainer(t *testing.T) *daggerkit.MockDaggerContainer {
	t.Helper()
	c := &daggerkit.MockDaggerContainer{}
	c.On("From", mock.Anything).Return(c)
	c.On("WithMountedCache", mock.Anything, mock.Anything).Return(c)
	c.On("WithMountedDirectory", mock.Anything, mock.Anything).Return(c)
	c.On("WithWorkdir", mock.Anything).Return(c)
	return c
}

// TestRustBuilder_Build_MockClient_Success exercises the same business
// logic as TestRustBuilder_Build_RealEngine (integration_test.go): a
// configured client builds the release binary and the returned Directory is
// the one the container chain produced -- but via daggerkit's mocks
// instead of a real Dagger engine, per the testify-only unit test mandate.
func TestRustBuilder_Build_MockClient_Success(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := mockBuildContainer(t)
	container.On("WithExec", []string{"cargo", "build", "--release"}).Return(container)
	container.On("WithExec", []string{"mkdir", "-p", "/output"}).Return(container)
	container.On("WithExec", []string{"cp", "/app/target/release/capabilitiestest", "/output/capabilitiestest"}).Return(container)
	container.On("Sync", mock.Anything).Return(container, nil)

	realDir := &dagger.Directory{}
	outputDir := &daggerkit.MockDaggerDirectory{}
	outputDir.On("GetRealDirectory").Return(realDir)
	container.On("Directory", "/output").Return(outputDir)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	builder := &rust.RustBuilder{
		Client: client,
		Config: shipwright.BuildConfig{BinaryName: "capabilitiestest"},
	}

	src := &dagger.Directory{}
	out, err := builder.Build(context.Background(), src)

	require.NoError(t, err)
	require.Same(t, realDir, out)
	container.AssertExpectations(t)
}

// TestRustBuilder_Build_MockClient_CompileErrorIncludesStderr mirrors
// TestRustBuilder_Build_RealEngine_CompileErrorIncludesStderr: a
// *dagger.ExecError surfacing from Sync must be expanded via wrapExecError
// so the real rustc diagnostics (stderr) reach the returned error, instead
// of dagger.ExecError.Error()'s generic message.
func TestRustBuilder_Build_MockClient_CompileErrorIncludesStderr(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := mockBuildContainer(t)
	container.On("WithExec", mock.Anything).Return(container)

	execErr := &dagger.ExecError{
		ExitCode: 101,
		Stderr:   "error[E0308]: mismatched types\n",
	}
	container.On("Sync", mock.Anything).Return(nil, execErr)

	client.On("Container").Return(container)
	client.On("CacheVolume", mock.Anything).Return(&daggerkit.MockDaggerCacheVolume{})

	builder := &rust.RustBuilder{
		Client: client,
		Config: shipwright.BuildConfig{BinaryName: "brokencrate"},
	}

	_, err := builder.Build(context.Background(), &dagger.Directory{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "stderr:")
	require.Contains(t, err.Error(), "mismatched types")
}

// TestRustBuilder_Build_MockClient_RepeatedBuildReusesCache mirrors
// TestRustBuilder_Build_RealEngine_RepeatedBuildReusesCache: it proves the
// same two cache-volume keys (cargo registry + build target) are mounted on
// every call to Build against the same client, not just the first.
func TestRustBuilder_Build_MockClient_RepeatedBuildReusesCache(t *testing.T) {
	client := &daggerkit.MockDaggerClient{}
	container := mockBuildContainer(t)
	container.On("WithExec", mock.Anything).Return(container)
	container.On("Sync", mock.Anything).Return(container, nil)

	realDir := &dagger.Directory{}
	outputDir := &daggerkit.MockDaggerDirectory{}
	outputDir.On("GetRealDirectory").Return(realDir)
	container.On("Directory", "/output").Return(outputDir)

	client.On("Container").Return(container)
	client.On("CacheVolume", "shipwright-cargo-registry").Return(&daggerkit.MockDaggerCacheVolume{})
	client.On("CacheVolume", "shipwright-rust-builder-target").Return(&daggerkit.MockDaggerCacheVolume{})

	builder := &rust.RustBuilder{
		Client: client,
		Config: shipwright.BuildConfig{BinaryName: "cachetest"},
	}

	src := &dagger.Directory{}
	for i := 0; i < 2; i++ {
		out, err := builder.Build(context.Background(), src)
		require.NoErrorf(t, err, "call #%d", i+1)
		require.Samef(t, realDir, out, "call #%d", i+1)
	}

	client.AssertNumberOfCalls(t, "CacheVolume", 4)
	client.AssertCalled(t, "CacheVolume", "shipwright-cargo-registry")
	client.AssertCalled(t, "CacheVolume", "shipwright-rust-builder-target")
}
