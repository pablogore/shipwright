package rust

import (
	"context"
	"errors"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// defaultDockerSocketPath is the conventional Unix socket path a Docker
// daemon listens on, both on a typical CI runner host and inside the
// container this capability mounts it into.
const defaultDockerSocketPath = "/var/run/docker.sock"

// RustIntegrationTester runs a Rust workspace's service-dependent
// integration suite (e.g. ego-rs's integration-tests/Cargo.toml, built on
// testcontainers + testcontainers-modules against real PostgreSQL) and
// returns the captured test output as the report File.
//
// Registered under the "test" capability (RegisterTester), NOT a new
// "integration-test" capability kind: internal/workflow/manifest/
// validate.go's validCapabilities is a fixed five-entry allowlist mirroring
// pkg/shipwright's five public Layer 1 interfaces (design.md D-H, "capability
// Is The Contract, uses Is The Implementation") — adding a sixth kind would
// mean amending that versioned public contract, not just this provider.
// Coexisting as another Tester alongside RustUnitTester/RustLinter/
// RustVulnScanner under the same "test" capability, distinguished only by
// provider name ("rust-integration-test"), gets the same practical
// outcome — a separate workflow step — without touching Layer 1.
//
// Docker access, compatibility mode: mounts the CI host's own Docker socket
// into the test container (Docker-outside-of-Docker) rather than running a
// nested dockerd, so testcontainers-rs (via bollard) talks to a real running
// daemon with no changes to ego-rs's existing integration-tests suite.
// Requires the host actually running Dagger to have Docker available at
// DockerSocketPath — true of GitHub-hosted runners and most self-hosted CI,
// but a real precondition this capability does not itself provision. A
// later iteration replacing this with Dagger-native `services:` bindings
// (Postgres as a service dependency, no Docker socket at all) does not
// change this Tester's public shape.
type RustIntegrationTester struct {
	// Client is the Dagger client used to construct the test container.
	Client *dagger.Client
	// RustVersion selects the Rust toolchain image. Defaults to
	// defaultRustVersion when left empty, same convention as
	// RustUnitTester.RustVersion.
	RustVersion string
	// ManifestPath selects the integration workspace's own Cargo.toml
	// (e.g. "integration-tests/Cargo.toml"), distinct from the main
	// workspace cargo test never touches.
	ManifestPath string
	// Package restricts the run to one member of the integration
	// workspace via `cargo test --package`. Leaving it empty runs
	// `--workspace` against ManifestPath instead.
	Package string
	// Features lists the Cargo features to enable via `--features` —
	// e.g. ego-rs's own crash-test-failpoint, which its Cargo.toml
	// documents as belonging only inside this isolated workspace.
	// Ignored when AllFeatures is set.
	Features []string
	// AllFeatures maps to `--all-features`.
	AllFeatures bool
	// Locked maps to `--locked`.
	Locked bool
	// DockerSocketPath overrides the host Docker socket path. Defaults to
	// defaultDockerSocketPath when left empty.
	DockerSocketPath string
}

// Compile-time conformance assertion: RustIntegrationTester must satisfy
// Layer 1's Tester interface.
var _ shipwright.Tester = (*RustIntegrationTester)(nil)

// Test runs the source Directory's integration suite (ManifestPath's
// workspace) with the host's Docker socket mounted in, and returns the
// captured test output as the report File.
func (t *RustIntegrationTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if t.Client == nil {
		return nil, errors.New("rustintegrationtester: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustintegrationtester: source directory is nil")
	}

	rustVersion := resolveRustVersion(t.RustVersion)
	socketPath := resolveDockerSocketPath(t.DockerSocketPath)
	dockerSocket := t.Client.Host().UnixSocket(socketPath)

	container := t.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, t.Client.CacheVolume(cargoRegistryCacheKey)).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/src/target", t.Client.CacheVolume(rustIntegrationTesterTargetCacheKey)).
		WithUnixSocket(socketPath, dockerSocket).
		WithExec(t.cargoTestArgs())

	testOutput, err := container.Stdout(ctx)
	if err != nil {
		return nil, wrapExecError("rustintegrationtester: integration tests failed", err)
	}

	reportContainer := container.WithNewFile("/tmp/integration-test-output.txt", testOutput)
	return reportContainer.File("/tmp/integration-test-output.txt"), nil
}

// cargoTestArgs builds the `cargo test` invocation from t's configuration.
// See cargoTestArgsFor (cargotestargs.go) for the shared selection logic.
func (t *RustIntegrationTester) cargoTestArgs() []string {
	return cargoTestArgsFor(t.ManifestPath, t.Package, t.Locked, t.AllFeatures, t.Features)
}

// resolveDockerSocketPath returns cfgPath, or defaultDockerSocketPath when
// cfgPath is empty. Extracted as a pure helper so it is unit-testable
// without a Dagger client.
func resolveDockerSocketPath(cfgPath string) string {
	if cfgPath == "" {
		return defaultDockerSocketPath
	}
	return cfgPath
}
