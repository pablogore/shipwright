package rust

import (
	"context"
	"errors"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

// RustLinter runs cargo clippy against a source Directory and returns its
// output as the report File. Structural mirror of providers/go's GoLinter —
// one of three independent Tester implementations, none privileged, for
// the Rust toolchain.
//nolint:revive // stutters with package rust by design: this is a deliberate structural mirror of providers/go's GoLinter (see doc comment above), and every rust.Rust* type follows the same cross-provider naming symmetry
type RustLinter struct {
	// Client is the Dagger client used to construct the lint container.
	Client daggerkit.DaggerClient
	// RustVersion selects the Rust toolchain image tag. Unlike
	// golangci-lint (a standalone binary distributed in its own image),
	// clippy is a rustup component bound to a specific toolchain, so this
	// implementation needs its own version knob. Defaults to
	// defaultRustVersion when left empty.
	RustVersion string
}

// Compile-time conformance assertion: RustLinter must satisfy Layer 1's
// Tester interface.
var _ shipwright.Tester = (*RustLinter)(nil)

// Test runs cargo clippy against the source Directory, treating every
// warning as an error (`-- -D warnings`, the direct clippy analog of
// golangci-lint's default fail-on-issues behavior), and returns its
// captured stdout as the report File.
func (l *RustLinter) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if l.Client == nil {
		return nil, errors.New("rustlinter: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustlinter: source directory is nil")
	}

	rustVersion := resolveRustVersion(l.RustVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	// rustup component add clippy runs against the base image only — before
	// the source directory is mounted — so its BuildKit cache entry is keyed
	// solely on the rust:<version> image and the registry cache mount
	// identity, never on source content. Measured effect: mounting source
	// first (the original ordering) forced this step to be recomputed on
	// every source change even though it has zero dependency on source
	// content, costing on the order of several hundred ms per run for no
	// reason; this ordering makes the step's cache genuinely stable across
	// every run of the same Rust version, cold or warm.
	toolchain := l.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, l.Client.CacheVolume(cargoRegistryCacheKey)).
		WithExec([]string{"rustup", "component", "add", "clippy"})

	container := toolchain.
		WithMountedDirectory("/app", sourceDir).
		WithWorkdir("/app").
		WithMountedCache("/app/target", l.Client.CacheVolume(rustLinterTargetCacheKey)).
		// cargo fetch splits the network-bound dependency download out of the
		// clippy exec below. Dagger's progress tree (and `dagger run`'s CLI
		// output) shows one row per WithExec with its own final duration, but
		// never streams a running exec's stdout/stderr live — so a single
		// combined "install clippy + fetch deps + compile + lint" exec looks
		// identical (silent, no visible progress) whether it's stuck
		// downloading crates or genuinely still linting a large workspace.
		// Isolating the fetch here costs nothing (cargo clippy would run it
		// implicitly anyway) and turns that ambiguity into two distinct,
		// individually-timed steps, without changing the final clippy
		// invocation itself.
		WithExec([]string{"cargo", "fetch"})

	lintContainer := container.WithExec([]string{"cargo", "clippy", "--all-targets", "--", "-D", "warnings"})

	stdout, err := lintContainer.Stdout(ctx)
	if err != nil {
		return nil, wrapExecError("rustlinter: cargo clippy found issues", err)
	}
	// clippy, like rustc, writes its actual diagnostics (warnings/errors) to
	// stderr rather than stdout — capturing only Stdout above left the
	// report file (and, before wrapExecError, the returned error) without
	// clippy's real diagnostic detail.
	stderr, err := lintContainer.Stderr(ctx)
	if err != nil {
		return nil, wrapExecError("rustlinter: cargo clippy found issues", err)
	}

	report := stdout
	if stderr != "" {
		report += "\n" + stderr
	}

	reportContainer := container.WithNewFile("/tmp/lint-report.txt", report)
	return reportContainer.File("/tmp/lint-report.txt").GetRealFile(), nil
}
