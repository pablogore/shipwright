package golang

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// defaultLinterImage matches the legacy pipeline's golangci-lint image
// (internal/pipelines/go-service/pipeline.go's Lint method).
const defaultLinterImage = "golangci/golangci-lint:latest"

// defaultLinterTimeout matches the legacy pipeline's hardcoded lint
// timeout.
const defaultLinterTimeout = "5m"

// GoLinter runs golangci-lint against a source Directory and returns its
// output as the report File. Extracted from the legacy go-service
// pipeline's Lint logic (internal/pipelines/go-service/pipeline.go) — one
// of three independent Tester implementations produced by the go-service
// decomposition (design.md D-F).
type GoLinter struct {
	// Client is the Dagger client used to construct the lint container.
	Client *dagger.Client
}

// Compile-time conformance assertion (tasks.md 3.5): GoLinter must satisfy
// Layer 1's Tester interface.
var _ shipwright.Tester = (*GoLinter)(nil)

// Test runs golangci-lint against the source Directory and returns its
// captured stdout as the report File. golangci-lint's own .golangci.yml
// configuration, if present in source, is picked up automatically —
// identical to the legacy behavior.
func (l *GoLinter) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if l.Client == nil {
		return nil, errors.New("golinter: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("golinter: source directory is nil")
	}

	container := l.Client.Container().
		From(defaultLinterImage).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("CGO_ENABLED", "0")

	output, err := container.
		WithExec([]string{"golangci-lint", "run", "--timeout", defaultLinterTimeout, "./..."}).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("golinter: golangci-lint found issues: %w", err)
	}

	reportContainer := container.WithNewFile("/tmp/lint-report.txt", output)
	return reportContainer.File("/tmp/lint-report.txt"), nil
}
