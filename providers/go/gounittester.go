package golang

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// coverageTotalRegexp matches `go tool cover -func`'s total-coverage line,
// identical to the pattern used by the legacy pipeline
// (internal/pipelines/shared.RunTestsWithCoverage).
var coverageTotalRegexp = regexp.MustCompile(`total:\s+\(statements\)\s+(\d+\.\d+)%`)

// GoUnitTester runs `go test` with coverage and race detection against a
// source Directory and returns the coverage profile as the report File.
// Extracted from the legacy go-service pipeline's Test logic
// (internal/pipelines/go-service/pipeline.go, via
// internal/pipelines/shared.RunTestsWithCoverage) — one of three
// independent Tester implementations produced by the go-service
// decomposition (design.md D-F's orthogonality win: GoUnitTester,
// GoLinter, GoVulnScanner each register separately for capability: test).
type GoUnitTester struct {
	// Client is the Dagger client used to construct the test container.
	Client *dagger.Client
	// Config carries the minimum required coverage percentage.
	Config shipwright.TestConfig
	// GoVersion selects the Go toolchain image. Kept as its own field
	// rather than reusing BuildConfig, because TestConfig intentionally
	// has no GoVersion field (design.md D-D). Defaults to
	// defaultGoVersion when left empty.
	GoVersion string
}

// Compile-time conformance assertion (tasks.md 3.5): GoUnitTester must
// satisfy Layer 1's Tester interface.
var _ shipwright.Tester = (*GoUnitTester)(nil)

// Test runs the source Directory's Go test suite with coverage and race
// detection, enforcing Config.Coverage as a minimum threshold when set,
// and returns the raw coverage profile as the report File.
func (t *GoUnitTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if t.Client == nil {
		return nil, errors.New("gounittester: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("gounittester: source directory is nil")
	}

	goVersion := resolveGoVersion(t.GoVersion)

	container := t.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("GO111MODULE", "on").
		// CGO_ENABLED=1, not 0: `go test -race` requires cgo. The legacy
		// go-service pipeline set CGO_ENABLED=0 alongside -race
		// (internal/pipelines/shared.RunTestsWithCoverage) — a latent bug
		// never triggered because the legacy Pipeline.Test path is fully
		// mocked in its own test suite. Surfaced and fixed here via this
		// package's real-engine integration test.
		WithEnvVariable("CGO_ENABLED", "1").
		WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=/tmp/coverage.out", "./..."})

	ran, err := container.Sync(ctx)
	if err != nil {
		return nil, fmt.Errorf("gounittester: tests failed: %w", err)
	}

	if t.Config.Coverage > 0 {
		if err := t.enforceCoverageThreshold(ctx, ran); err != nil {
			return nil, err
		}
	}

	return ran.File("/tmp/coverage.out"), nil
}

// enforceCoverageThreshold reads the coverage percentage from the already
// executed container and returns an error naming the shortfall when it is
// below Config.Coverage.
func (t *GoUnitTester) enforceCoverageThreshold(ctx context.Context, ran *dagger.Container) error {
	coverageOutput, err := ran.WithExec([]string{"go", "tool", "cover", "-func=/tmp/coverage.out"}).Stdout(ctx)
	if err != nil {
		return fmt.Errorf("gounittester: failed to compute coverage: %w", err)
	}

	pct, err := parseCoveragePercentage(coverageOutput)
	if err != nil {
		return fmt.Errorf("gounittester: %w", err)
	}

	if pct < t.Config.Coverage {
		return fmt.Errorf("gounittester: coverage %.2f%% is below the required threshold of %.2f%%", pct, t.Config.Coverage)
	}

	return nil
}

// parseCoveragePercentage extracts the total statement-coverage percentage
// from `go tool cover -func` output. Pure helper, unit-testable without a
// Dagger client.
func parseCoveragePercentage(output string) (float64, error) {
	matches := coverageTotalRegexp.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("failed to parse coverage output: %s", output)
	}
	return strconv.ParseFloat(matches[1], 64)
}
