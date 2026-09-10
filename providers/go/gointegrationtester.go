package golang

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// defaultIntegrationCommand is the invocation resolveIntegrationCommand
// falls back to when GoIntegrationTester.Command is left empty: the
// ordinary Go convention for a service-dependent suite gated behind a build
// tag (design.md D-2), mirroring `-tags=integration`'s use throughout this
// repo's own testing/integration/ tree.
const defaultIntegrationCommand = "go test -tags=integration ./..."

// GoIntegrationTester runs a Go module's service-dependent integration
// suite (e.g. a testcontainers-go-backed suite gated behind a build tag)
// with a Docker-in-Docker daemon attached, and returns the captured test
// output as the report File.
//
// Registered under the "test" capability (RegisterTester), NOT a new
// "integration-test" capability kind — mirrors
// providers/rust/rustintegrationtester.go's own doc comment: coexisting as
// another Tester alongside GoUnitTester/GoLinter/GoVulnScanner/
// GoBuildChecker under the same "test" capability, distinguished only by
// provider name ("go-integration-test"), gets the same practical outcome (a
// separate workflow step) without amending Layer 1's fixed five-entry
// capability allowlist.
//
// Docker access is inherent to this capability and always attached: a
// privileged Docker-in-Docker daemon (see dockerdaemon.go) is bound into
// the test container so testcontainers-go talks to a real,
// network-reachable Docker daemon, with no changes required to the
// consumer repo's own integration suite (design.md D-9's "zero changes to
// the consumer repo's test code" success criterion).
type GoIntegrationTester struct {
	// Client is the Dagger client used to construct the test container.
	Client daggerkit.DaggerClient
	// GoVersion selects the Go toolchain image. Defaults to
	// defaultGoVersion when left empty, same convention as every other
	// Tester in this package.
	GoVersion string
	// Command is the shell command line to run in place of the default
	// `go test -tags=integration ./...` (design.md D-2). Combines
	// rust-integration-test's zero-config default with rust-command's
	// arbitrary-invocation reach: leave it empty for the common case, or
	// set it to something like "go build && ./inttest-runner" when the
	// suite needs more than a single `go test` invocation.
	//
	// Unlike RustCommand.Command (whitespace-tokenized argv, never a
	// shell — see rustcommand.go's own doc comment), Command here is run
	// via `sh -c` (design.md D-3): the motivating invocation shape
	// (`go build && ./inttest-runner`) is unrepresentable as argv, and
	// unlike RustCommand's contractually-a-cargo-subcommand-list field,
	// Command here is contractually a shell command line — the whole
	// string is the manifest author's declared program, nothing
	// Shipwright-controlled is concatenated into it, so no value can be
	// injected into the shell string. See the Threat Matrix section of
	// design.md for the full trust-boundary rationale.
	Command string
}

// Compile-time conformance assertion: GoIntegrationTester must satisfy
// shipwright.Tester.
var _ shipwright.Tester = (*GoIntegrationTester)(nil)

// Test runs the source Directory's integration suite (resolveIntegrationCommand's
// resolved Command, via `sh -c`) with a Docker-in-Docker daemon attached,
// and returns the captured test output as the report File.
func (t *GoIntegrationTester) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if t.Client == nil {
		return nil, errors.New("gointegrationtester: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("gointegrationtester: source directory is nil")
	}

	goVersion := resolveGoVersion(t.GoVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	container := t.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/src", sourceDir).
		WithWorkdir("/src").
		WithEnvVariable("GO111MODULE", "on") // no CGO_ENABLED — design.md D-8: Command is arbitrary, pinning either value would silently override the caller's own build settings
	container = mountGoCaches(t.Client, container)
	container = withDockerDaemon(ctx, t.Client, container)
	// TESTCONTAINERS_RYUK_DISABLED=true is a provider default, not a
	// consumer-configured one (design.md D-9): the DinD instance's lifetime
	// is exactly this step's lifetime, so everything Ryuk would reap dies
	// with the service anyway, and Ryuk's reap-on-disconnect handshake is
	// the single most common DinD-CI failure mode. Most likely of every
	// decision in design.md to be revisited once a real CI run reports
	// evidence either way; still overridable from inside Command.
	container = container.WithEnvVariable("TESTCONTAINERS_RYUK_DISABLED", "true")

	container = container.WithExec([]string{"sh", "-c", resolveIntegrationCommand(t.Command)}, daggerkit.DaggerContainerWithExecOpts{})

	testOutput, err := container.Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("gointegrationtester: integration tests failed: %w", err)
	}

	reportContainer := container.WithNewFile("/tmp/integration-test-output.txt", testOutput)
	return reportContainer.File("/tmp/integration-test-output.txt").GetRealFile(), nil
}

// resolveIntegrationCommand returns cmd, or defaultIntegrationCommand when
// cmd is empty — same shape as resolveGoVersion/resolveBinaryName
// (gobuilder.go).
func resolveIntegrationCommand(cmd string) string {
	if cmd == "" {
		return defaultIntegrationCommand
	}
	return cmd
}
