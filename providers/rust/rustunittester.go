package rust

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
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
	Client *dagger.Client
	// Config carries the minimum required test coverage percentage.
	Config shipwright.TestConfig
	// RustVersion selects the Rust toolchain image. Kept as its own field
	// rather than reusing BuildConfig, mirroring GoUnitTester.GoVersion's
	// own rationale (TestConfig intentionally has no toolchain-version
	// field). Defaults to defaultRustVersion when left empty.
	RustVersion string
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

	container := t.Client.Container().
		From("rust:"+rustVersion).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"cargo", "test", "--workspace", "--all-features"})

	testOutput, err := container.Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("rustunittester: tests failed: %w", err)
	}

	if t.Config.Coverage > 0 {
		if err := t.enforceCoverageThreshold(ctx, container); err != nil {
			return nil, err
		}
	}

	reportContainer := container.WithNewFile("/tmp/test-output.txt", testOutput)
	return reportContainer.File("/tmp/test-output.txt"), nil
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
func (t *RustUnitTester) enforceCoverageThreshold(ctx context.Context, container *dagger.Container) error {
	coverageOutput, err := container.
		WithExec([]string{"cargo", "install", "cargo-tarpaulin", "--locked"}).
		WithExec([]string{"cargo", "tarpaulin", "--workspace", "--out", "Stdout"}).
		Stdout(ctx)
	if err != nil {
		return fmt.Errorf("rustunittester: failed to compute coverage: %w", err)
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
