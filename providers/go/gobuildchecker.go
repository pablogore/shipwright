package golang

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// GoBuildChecker runs `go build ./...` against a source Directory as a
// compile gate for Go modules with no root `main` package (library-only
// modules) — GoBuilder's Builder contract requires a root `main` package
// and produces a binary Directory output, which a library module has
// neither the need for nor the shape to satisfy. GoBuildChecker is
// registered as a Tester (`capability: test`) instead: it operates
// directly on the workflow's source input, needs no companion `build`
// step, and its success/failure is a pass/fail gate, not an artifact
// pipeline stage (design.md, closes issue #273).
//
// This is deliberately NOT a vet/lint gate: golangci-lint (GoLinter,
// registered as "golangci-lint") already runs `go vet` as part of its own
// analyzer set, and `go build ./...` alone would duplicate that partially
// and incompletely. GoBuildChecker's only job is confirming the module
// actually compiles.
type GoBuildChecker struct {
	// Client is the Dagger client used to construct the build container.
	Client daggerkit.DaggerClient
	// GoVersion selects the Go toolchain image used to run `go build`.
	// Defaults to defaultGoVersion when left empty.
	GoVersion string
}

// Compile-time conformance assertion: GoBuildChecker must satisfy
// shipwright.Tester.
var _ shipwright.Tester = (*GoBuildChecker)(nil)

// Test runs `go build ./...` against the mounted source Directory. It
// does NOT run `go mod tidy` first (design.md D-3): tidying would mutate
// go.mod/go.sum in-container and could silently paper over a broken
// module graph — exactly the failure this compile gate exists to catch.
//
// On success, `go build ./...` prints nothing, so the report File's body
// is synthesized rather than passed through verbatim (design.md D-5). On
// failure, the wrapped error is returned as-is: Dagger's ExecError already
// embeds both stdout and stderr in its Error() text (the same behavior
// GoVulnScanner's own doc comment relies on), so no second exec or
// explicit Stderr(ctx) call is needed to surface the compiler's
// diagnostic output.
func (c *GoBuildChecker) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if c.Client == nil {
		return nil, errors.New("gobuildchecker: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("gobuildchecker: source directory is nil")
	}

	goVersion := resolveGoVersion(c.GoVersion)

	container := c.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/app", daggerkit.NewDaggerDirectoryAdapter(source)).
		WithWorkdir("/app").
		WithEnvVariable("GO111MODULE", "on").
		WithEnvVariable("CGO_ENABLED", "0")
	container = mountGoCaches(c.Client, container)

	output, err := container.WithExec([]string{"go", "build", "./..."}, daggerkit.DaggerContainerWithExecOpts{}).Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("gobuildchecker: go build ./... failed: %w", err)
	}

	report := fmt.Sprintf("go build ./... succeeded (golang:%s)\n%s", goVersion, output)
	reportContainer := container.WithNewFile("/tmp/build-check-report.txt", report)
	return reportContainer.File("/tmp/build-check-report.txt").GetRealFile(), nil
}
