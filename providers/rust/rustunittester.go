package rust

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/rust/daggerkit"
)

// tarpaulinCoverageRegexp matches cargo-tarpaulin's `--out Stdout` summary
// line ("87.50% coverage, 10/12 lines covered"), the closest Rust analog to
// `go tool cover -func`'s total-coverage line.
var tarpaulinCoverageRegexp = regexp.MustCompile(`(\d+\.\d+)% coverage`)

// RustUnitTester runs `cargo test` against a source Directory and returns
// the captured test output as the report File, optionally enforcing a
// minimum coverage threshold via cargo-tarpaulin. Structural mirror of
// providers/go's GoUnitTester — one of three independent Tester
// implementations, none privileged, for the Rust toolchain.
//
// -race analog: not applicable. Rust's ownership and borrow checker
// statically rule out data races in safe code at compile time — the same
// guarantee `go test -race` checks dynamically at runtime — so no
// race-detector flag is threaded through cargo test.
type RustUnitTester struct {
	// Client is the Dagger client used to construct the test container.
	Client daggerkit.DaggerClient
	// Config carries the minimum required test coverage percentage.
	Config shipwright.TestConfig
	// RustVersion selects the Rust toolchain image. Kept as its own field
	// rather than reusing BuildConfig, mirroring GoUnitTester.GoVersion's
	// own rationale (TestConfig intentionally has no toolchain-version
	// field). Defaults to defaultRustVersion when left empty.
	RustVersion string
	// ManifestPath selects a Cargo.toml other than the source root's own,
	// mirroring `cargo test --manifest-path`. Needed for a workspace like
	// ego-rs's, which keeps its Testcontainers-backed integration suite in
	// a deliberately separate, non-member workspace
	// (integration-tests/Cargo.toml) precisely so `cargo test --workspace`
	// never touches Docker.
	ManifestPath string
	// Package restricts the run to one workspace member (`cargo test
	// --package`) instead of the whole workspace. Mutually exclusive with
	// --workspace in practice, so set it leaves --workspace off entirely
	// (see cargoTestArgs).
	Package string
	// Features lists the Cargo features to enable via `--features`.
	// Ignored when AllFeatures is set.
	Features []string
	// AllFeatures maps to `--all-features`. Left false by default
	// (previously always forced on) because a provider must not decide a
	// consumer's feature policy: ego-rs's own crash-test-failpoint feature
	// is documented to stay off outside its isolated integration-tests
	// workspace, which --all-features would silently violate.
	AllFeatures bool
	// Locked maps to `--locked`, forbidding Cargo from updating
	// Cargo.lock so the same lockfile always produces the same dependency
	// graph in CI.
	Locked bool
}

// Compile-time conformance assertion: RustUnitTester must satisfy Layer 1's
// Tester interface.
var _ shipwright.Tester = (*RustUnitTester)(nil)

// Test runs the source Directory's cargo test suite, enforcing
// Config.Coverage as a minimum threshold when set, and returns the captured
// test output as the report File.
func (t *RustUnitTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if t.Client == nil {
		return nil, errors.New("rustunittester: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("rustunittester: source directory is nil")
	}

	rustVersion := resolveRustVersion(t.RustVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	container := t.Client.Container().
		From("rust:"+rustVersion).
		WithMountedCache(cargoRegistryMountPath, t.Client.CacheVolume(cargoRegistryCacheKey)).
		WithMountedDirectory("/src", sourceDir).
		WithWorkdir("/src").
		WithMountedCache("/src/target", t.Client.CacheVolume(rustUnitTesterTargetCacheKey)).
		WithExec(t.cargoTestArgs())

	testOutput, err := container.Stdout(ctx)
	if err != nil {
		return nil, wrapExecError("rustunittester: tests failed", err)
	}

	if t.Config.Coverage > 0 {
		if err := t.enforceCoverageThreshold(ctx, container); err != nil {
			return nil, err
		}
	}

	reportContainer := container.WithNewFile("/tmp/test-output.txt", testOutput)
	return reportContainer.File("/tmp/test-output.txt").GetRealFile(), nil
}

// cargoTestArgs builds the `cargo test` invocation from t's configuration.
// See cargoTestArgsFor (cargotestargs.go) for the shared selection logic.
func (t *RustUnitTester) cargoTestArgs() []string {
	return cargoTestArgsFor(t.ManifestPath, t.Package, t.Locked, t.AllFeatures, t.Features)
}

// tarpaulinArgs builds the `cargo tarpaulin` invocation from t's
// configuration, sharing cargoScopeArgsFor with cargoTestArgs so coverage is
// always measured over exactly the scope that was tested.
func (t *RustUnitTester) tarpaulinArgs() []string {
	return append([]string{"cargo", "tarpaulin", "--out", "Stdout", "--engine", "llvm"},
		cargoScopeArgsFor(t.ManifestPath, t.Package, t.Locked, t.AllFeatures, t.Features)...)
}

// enforceCoverageThreshold installs cargo-tarpaulin and runs it against the
// already-tested container, returning an error naming the shortfall when
// coverage is below Config.Coverage.
//
// Design decision: cargo-tarpaulin over cargo-llvm-cov. Both are
// widely-used Rust coverage tools, but llvm-cov's `--summary-only` output
// is a multi-column table (regions/functions/lines/branches, each with its
// own percentage) that would require fragile column-position parsing to
// extract a single number. tarpaulin's `--out Stdout` prints one
// unambiguous "NN.NN% coverage" summary line, the direct structural analog
// of `go tool cover -func`'s single total line that
// coverageTotalRegexp/parseCoveragePercentage already parse in
// providers/go.
//
// --engine llvm: switches tarpaulin to LLVM source-based instrumentation
// instead of its default ptrace-based engine, which requires the
// SYS_PTRACE capability Dagger's sandboxed exec doesn't grant. Kept as a
// defensive default even though it was NOT the cause of the CI failure
// below (ptrace was the original hypothesis before wrapExecError surfaced
// the real stderr) — ptrace-in-containers remains a real, separate failure
// mode this avoids.
//
// The actual CI failure (exit code 101) was cargo-tarpaulin's own install
// step: `cargo install cargo-tarpaulin --locked` with no version pin always
// pulls the latest release, and tarpaulin 0.37.2's Cargo.toml requires the
// `edition2024` Cargo feature, which only stabilized in Rust 1.85.0 — the
// image this ran under was still on the older defaultRustVersion
// ("1.83.0"). Fixed by bumping defaultRustVersion (rustbuilder.go) to
// 1.85.0 and pinning --version below, so a future tarpaulin release
// bumping its own MSRV/edition again can't silently break this the same
// way.
func (t *RustUnitTester) enforceCoverageThreshold(ctx context.Context, container daggerkit.DaggerContainer) error {
	coverageOutput, err := container.
		WithExec([]string{"cargo", "install", "cargo-tarpaulin", "--locked", "--version", "0.37.2"}).
		WithExec(t.tarpaulinArgs()).
		Stdout(ctx)
	if err != nil {
		return wrapExecError("rustunittester: failed to compute coverage", err)
	}

	pct, err := parseTarpaulinCoverage(coverageOutput)
	if err != nil {
		return fmt.Errorf("rustunittester: %w", err)
	}

	if pct < t.Config.Coverage {
		return fmt.Errorf("rustunittester: coverage %.2f%% is below the required threshold of %.2f%%", pct, t.Config.Coverage)
	}

	return nil
}

// wrapExecError formats err as a prefixed error, expanding a
// *dagger.ExecError into its captured exit code and stderr rather than
// leaving a bare `%w`-wrapped message — the blind spot that made
// TestRustUnitTester_Test_RealEngine_PassesWithinThreshold's CI failure
// ("rustunittester: failed to compute coverage: exit code: 101")
// undiagnosable from the error text alone, since
// dagger.ExecError.Error()/.Message() carries only the generic "process
// ... did not complete successfully" text and never the command's actual
// stderr output.
//
// Deliberately reads only ExecError's exported fields (ExitCode, Stderr)
// rather than calling its Error()/Message() methods: those methods defer
// to an unexported `original` field that only Dagger's own error
// construction ever populates, so calling them on a synthetically built
// *dagger.ExecError (as a unit test must, since that field can't be set
// from outside the dagger package) would panic on a nil interface. Reading
// only the exported fields keeps this helper safe to unit-test without a
// live Dagger client, while behaving identically against a real one.
func wrapExecError(prefix string, err error) error {
	var execErr *dagger.ExecError
	if errors.As(err, &execErr) {
		return fmt.Errorf("%s: exit code %d: stderr: %s", prefix, execErr.ExitCode, strings.TrimSpace(execErr.Stderr))
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// parseTarpaulinCoverage extracts the total line-coverage percentage from
// cargo-tarpaulin's `--out Stdout` output. Pure helper, unit-testable
// without a Dagger client.
func parseTarpaulinCoverage(output string) (float64, error) {
	matches := tarpaulinCoverageRegexp.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("failed to parse coverage output: %s", output)
	}
	return strconv.ParseFloat(matches[1], 64)
}
