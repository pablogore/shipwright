package app

import (
	"testing"

	"dagger.io/dagger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	golang "github.com/pablogore/shipwright/providers/go"

	"github.com/pablogore/shipwright/internal/pipelines"
)

// TestBuildCapabilities_NilClientReturnsEmpty proves BuildCapabilities never
// constructs Layer 1 capability implementations against a nil Dagger
// client (WU10, tasks.md 10.2/10.3) — matching Capabilities' "any field MAY
// be nil" invariant rather than panicking or returning half-built
// implementations that would fail on first use.
func TestBuildCapabilities_NilClientReturnsEmpty(t *testing.T) {
	caps := BuildCapabilities(nil, pipelines.Config{GoVersion: "1.26.1"})

	assert.Nil(t, caps.Builder)
	assert.Nil(t, caps.Testers)
	assert.Nil(t, caps.Artifactor)
	assert.Nil(t, caps.Deployer)
	assert.Nil(t, caps.Runner)
}

// TestBuildCapabilities_WiresRealImplementations proves BuildCapabilities
// wires the standalone providers/go implementations (originally Phase 3,
// extracted from internal/capabilities), carrying the legacy
// pipelines.Config values through to their per-
// capability shipwright config structs (design.md D-D), when a non-nil
// Dagger client is available.
func TestBuildCapabilities_WiresRealImplementations(t *testing.T) {
	client := &dagger.Client{}
	cfg := pipelines.Config{
		GoVersion:    "1.26.1",
		JavaVersion:  "17",
		Coverage:     80.0,
		Registry:     "registry.example.com/org/svc",
		RegistryURL:  "registry.example.com",
		RegistryUser: "ci-token",
		ImageName:    "svc",
		ImageTag:     "v1.2.3",
		BuildTag:     "v1.2.3",
		CommitSHA:    "abc123",
		BranchName:   "main",
		Version:      "1.2.3",
		// RegistryPass/RegistryToken/Token deliberately left empty: their
		// non-empty path calls client.SetSecret, which requires a live
		// Dagger engine connection and is not exercised in this unit test
		// (matches this file's own pre-existing pattern for real-infra
		// paths — see WU9 apply-progress's executeWorkflow/
		// resolveWorkflowSecrets note).
	}

	caps := BuildCapabilities(client, cfg)

	require.NotNil(t, caps.Builder)
	builder, ok := caps.Builder.(*golang.GoBuilder)
	require.True(t, ok, "Builder must be *golang.GoBuilder")
	assert.Equal(t, client, builder.Client)
	assert.Equal(t, "1.26.1", builder.Config.GoVersion)
	assert.Equal(t, "17", builder.Config.JavaVersion)

	require.Len(t, caps.Testers, 3, "GoUnitTester, GoLinter, GoVulnScanner")
	unitTester, ok := caps.Testers[0].(*golang.GoUnitTester)
	require.True(t, ok, "first Tester must be *golang.GoUnitTester")
	assert.Equal(t, client, unitTester.Client)
	assert.InDelta(t, 80.0, unitTester.Config.Coverage, 0.0001)
	assert.Equal(t, "1.26.1", unitTester.GoVersion)

	linter, ok := caps.Testers[1].(*golang.GoLinter)
	require.True(t, ok, "second Tester must be *golang.GoLinter")
	assert.Equal(t, client, linter.Client)

	vulnScanner, ok := caps.Testers[2].(*golang.GoVulnScanner)
	require.True(t, ok, "third Tester must be *golang.GoVulnScanner")
	assert.Equal(t, client, vulnScanner.Client)
	assert.Equal(t, "1.26.1", vulnScanner.GoVersion)

	require.NotNil(t, caps.Artifactor)
	publisher, ok := caps.Artifactor.(*golang.ContainerPublisher)
	require.True(t, ok, "Artifactor must be *golang.ContainerPublisher")
	assert.Equal(t, client, publisher.Client)
	assert.Equal(t, "registry.example.com/org/svc", publisher.Config.Registry)
	assert.Equal(t, "registry.example.com", publisher.Config.RegistryURL)
	assert.Equal(t, "ci-token", publisher.Config.RegistryUser)
	assert.Equal(t, "svc", publisher.Config.ImageName)
	assert.Equal(t, "v1.2.3", publisher.Config.ImageTag)
	assert.Equal(t, "v1.2.3", publisher.Config.BuildTag)
	assert.Equal(t, "abc123", publisher.Config.CommitSHA)
	assert.Equal(t, "main", publisher.Config.BranchName)
	assert.Equal(t, "1.2.3", publisher.Config.Version)
	assert.Nil(t, publisher.Config.RegistryPass, "empty RegistryPass must not call SetSecret")
	assert.Nil(t, publisher.Config.RegistryToken, "empty RegistryToken must not call SetSecret")
	assert.Nil(t, publisher.Config.Token, "empty Token must not call SetSecret")

	// Deployer/Runner have no concrete implementation yet (design.md D-D):
	// always nil, never a partially-built stub.
	assert.Nil(t, caps.Deployer)
	assert.Nil(t, caps.Runner)
}
