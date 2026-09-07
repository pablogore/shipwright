package rust

import (
	"context"
	"errors"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

// RustCommand runs an arbitrary `cargo <args>` invocation against a source
// Directory and returns the captured output as the report File. It exists
// because RustBuilder/RustUnitTester/RustLinter/RustVulnScanner/
// RustIntegrationTester each encode one fixed Cargo invocation shape
// (`cargo build`, `cargo test`, `cargo clippy`, `cargo audit`) and cannot
// express an arbitrary subcommand like `cargo run -p xtask -- verify-layers`
// or a project-specific binary like `cargo run --bin run-suite` without a
// new, equally narrow Tester per invocation. RustCommand is the general
// primitive those are conveniences over — it does not replace them.
//
// Registered under the "test" capability (RegisterTester), coexisting with
// "rust-test"/"rust-integration-test"/"clippy"/"cargo-audit" — see
// rustintegrationtester.go's own doc comment for why a new Cargo invocation
// shape is a new provider name under the existing five-entry (now seven)
// capability allowlist, never a new capability kind.
//
// Deliberately NOT a shell executor: Command is tokenized on whitespace
// (cargoCommandArgsFor) and passed to Dagger's WithExec as argv, never
// through `sh -c`. A manifest author cannot smuggle in pipes, redirects,
// `&&`, or `$VAR` expansion this way — those characters simply become
// literal (almost always invalid) argv tokens to cargo, because no shell is
// ever invoked to interpret them. The trade-off: an argument containing a
// literal space (e.g. a quoted string) cannot be expressed in v1, since
// whitespace-splitting has no quoting support. Accepted because every known
// real invocation (`run -p xtask -- verify-layers`, `run --bin run-suite`,
// `run --manifest-path ... --bin run-suite`) space-delimits flags and
// values with no embedded-space arguments, and introducing a hand-rolled
// quote-aware tokenizer for a case with no known caller would be exactly
// the "fragile bespoke parser to avoid sh -c" this design explicitly
// avoids. A future version can add quoting support (or a list-typed
// manifest field, if one is ever added to interp.Kind) if a real need
// appears.
//nolint:revive // stutters with package rust by design: every rust.Rust* type names what it implements, matching this package's existing naming convention (see rustintegrationtester.go's own nolint comment)
type RustCommand struct {
	// Client is the Dagger client used to construct the command container.
	Client daggerkit.DaggerClient
	// RustVersion selects the Rust toolchain image. Defaults to
	// defaultRustVersion when left empty, same convention as every other
	// Tester in this package.
	RustVersion string
	// ManifestPath selects a Cargo.toml other than the source root's own.
	// The provider — not the caller — turns this into `--manifest-path`,
	// inserted right after the subcommand, so a manifest never duplicates
	// the same information inside Command (e.g. writing both
	// `manifestPath: integration-tests/Cargo.toml` and `--manifest-path
	// integration-tests/Cargo.toml` inside Command).
	ManifestPath string
	// Command is the Cargo invocation to run, without the leading "cargo"
	// (e.g. "run -p xtask -- verify-layers"). Required — Test rejects an
	// empty Command rather than running bare `cargo` with no subcommand.
	Command string
	// Docker opts into attaching a real Docker-in-Docker daemon (see
	// dockerdaemon.go) to the command container, giving it a
	// network-reachable DOCKER_HOST. Unlike RustIntegrationTester, where
	// Docker access is inherent to the capability and always attached, most
	// Cargo commands (xtask-style workspace tooling) must NOT get a Docker
	// daemon at all, so this defaults to false. Set it for a command that
	// needs one, e.g. a Testcontainers-backed run-suite binary.
	Docker bool
	// CacheKey isolates this step's `target` cache volume from every other
	// RustCommand step. Left empty, every RustCommand shares one default
	// cache volume — fine when a workflow only ever runs commands against
	// the same Cargo workspace. Two RustCommand steps against different
	// workspaces (e.g. the main workspace's xtask vs.
	// integration-tests/Cargo.toml's run-suite) must set distinct CacheKeys,
	// the same way rustIntegrationTesterTargetCacheKey is kept separate
	// from rustUnitTesterTargetCacheKey (cargocache.go) — otherwise
	// incremental-compilation state from one workspace corrupts the other's
	// target directory.
	CacheKey string
}

// Compile-time conformance assertion: RustCommand must satisfy Layer 1's
// Tester interface.
var _ shipwright.Tester = (*RustCommand)(nil)

// Test runs `cargo` + Command (plus --manifest-path when ManifestPath is
// set) against the source Directory, optionally with a Docker-in-Docker
// daemon attached, and returns the captured output as the report File.
// Pass/fail is exclusively the exit code Dagger reports for the exec: a
// non-nil error here always means the underlying cargo invocation failed
// (or the container never ran at all), never a partial or inferred result.
func (c *RustCommand) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if c.Client == nil {
		return nil, errors.New("rustcommand: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustcommand: source directory is nil")
	}
	if strings.TrimSpace(c.Command) == "" {
		return nil, errors.New("rustcommand: command is required")
	}

	rustVersion := resolveRustVersion(c.RustVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	container := c.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, c.Client.CacheVolume(cargoRegistryCacheKey)).
		WithMountedDirectory("/src", sourceDir).
		WithWorkdir("/src").
		WithMountedCache("/src/target", c.Client.CacheVolume(resolveCommandCacheKey(c.CacheKey)))

	if c.Docker {
		container = withDockerDaemon(c.Client, container)
	}

	container = container.WithExec(c.cargoCommandArgs())

	output, err := container.Stdout(ctx)
	if err != nil {
		return nil, wrapExecError("rustcommand: cargo command failed", err)
	}

	reportContainer := container.WithNewFile("/tmp/command-output.txt", output)
	return reportContainer.File("/tmp/command-output.txt").GetRealFile(), nil
}

// cargoCommandArgs builds the `cargo <args>` invocation from c's
// configuration. See cargoCommandArgsFor for the tokenization/manifest-path
// insertion logic, extracted as a pure helper so it is unit-testable
// without a Dagger client.
func (c *RustCommand) cargoCommandArgs() []string {
	return cargoCommandArgsFor(c.ManifestPath, c.Command)
}

// cargoCommandArgsFor tokenizes command on whitespace (no shell, no
// quoting — see RustCommand's own doc comment) and prepends "cargo",
// inserting "--manifest-path manifestPath" immediately after the
// subcommand when manifestPath is set — e.g.
// cargoCommandArgsFor("integration-tests/Cargo.toml", "run --bin run-suite")
// -> ["cargo", "run", "--manifest-path", "integration-tests/Cargo.toml",
// "--bin", "run-suite"], matching cargo's own convention of accepting
// --manifest-path right after the subcommand.
func cargoCommandArgsFor(manifestPath, command string) []string {
	tokens := strings.Fields(command)
	args := []string{"cargo"}
	if len(tokens) == 0 {
		return args
	}

	args = append(args, tokens[0])
	if manifestPath != "" {
		args = append(args, "--manifest-path", manifestPath)
	}
	return append(args, tokens[1:]...)
}

// resolveCommandCacheKey returns the shared default RustCommand target
// cache key, or a key namespaced under cacheKey when set — see
// RustCommand.CacheKey's own doc comment for why isolation matters.
func resolveCommandCacheKey(cacheKey string) string {
	if cacheKey == "" {
		return rustCommandDefaultTargetCacheKey
	}
	return rustCommandTargetCacheKeyPrefix + cacheKey
}
