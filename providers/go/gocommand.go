package golang

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
	"github.com/pablogore/shipwright/providers/go/daggerkit"
)

// GoCommand runs an arbitrary `go <args>` invocation against a source
// Directory and returns the captured output as the report File. It exists
// because GoBuildChecker/GoUnitTester/GoLinter/GoVulnScanner/
// GoIntegrationTester each encode one fixed invocation shape (`go build
// ./...`, `go test`, golangci-lint's analyzer set, govulncheck, an
// integration suite) and cannot express an arbitrary subcommand like
// `go build -o bin/api ./cmd/api` or `go generate ./...` without a new,
// equally narrow Tester per invocation. GoCommand is the general primitive
// those are conveniences over — it does not replace them. Go's analog of
// providers/rust/rustcommand.go's RustCommand; see that type's own doc
// comment for the shared design rationale.
//
// Registered under the "test" capability (RegisterTester), coexisting with
// "go-test"/"go-build"/"golangci-lint"/"govulncheck"/"go-integration-test"
// — a new `go` invocation shape is a new provider name under the existing
// capability allowlist, never a new capability kind (mirrors
// rustcommand.go's own doc comment on this point).
//
// Deliberately NOT a shell executor: Command is tokenized on whitespace
// (goCommandArgsFor) and passed to Dagger's WithExec as argv, never through
// `sh -c`. A manifest author cannot smuggle in pipes, redirects, `&&`, or
// `$VAR` expansion this way — those characters simply become literal
// (almost always invalid) argv tokens to go, because no shell is ever
// invoked to interpret them. The trade-off — an argument containing a
// literal space cannot be expressed in v1 — is the same one RustCommand
// accepts, for the same reason.
//
// Unlike RustCommand, GoCommand has no CacheKey field: gocache.go's shared
// GOMODCACHE/GOCACHE volumes are content-addressed by Go's own action ID
// (toolchain identity, GOOS/GOARCH, build flags all feed the hash), so the
// cross-workspace `target`-directory corruption RustCommand.CacheKey exists
// to prevent cannot occur here — two GoCommand steps safely share the same
// cache volumes regardless of WorkDir.
//
// WorkDir selects a subdirectory via `go -C <WorkDir>` (see the field's own
// doc comment for the exact chdir semantics), never a Docker with-field or
// Docker-in-Docker daemon — go-integration-test remains the sole Go Tester
// with Docker access.
type GoCommand struct {
	// Client is the Dagger client used to construct the command container.
	Client daggerkit.DaggerClient
	// GoVersion selects the Go toolchain image. Defaults to
	// defaultGoVersion when left empty, same convention as every other
	// Tester in this package.
	GoVersion string
	// WorkDir selects a subdirectory of the mounted source (relative to
	// /src) for the go invocation to run against. The provider — not the
	// caller — turns this into `go -C <WorkDir>`, inserted immediately
	// before the subcommand token (go requires -C to be argv[1]), so a
	// manifest never duplicates the same information inside Command.
	//
	// `-C` chdirs before the command runs, and any relative path named on
	// the command line is interpreted AFTER that chdir. So
	// WorkDir: "cmd/api", Command: "build -o bin/api ." writes
	// /src/cmd/api/bin/api, not /src/bin/api. This differs from
	// RustCommand.ManifestPath's `--manifest-path`, which does NOT chdir —
	// the one place this type is not a literal mirror of RustCommand.
	WorkDir string
	// Command is the go invocation to run, without the leading "go" (e.g.
	// "build -o bin/api ./cmd/api"). Required — Test rejects an empty (or
	// whitespace-only) Command rather than running bare `go` with no
	// subcommand.
	Command string
}

// Compile-time conformance assertion: GoCommand must satisfy Layer 1's
// Tester interface.
var _ shipwright.Tester = (*GoCommand)(nil)

// Test runs `go` + Command (with `-C WorkDir` inserted before the
// subcommand when WorkDir is set) against the source Directory and returns
// the captured output as the report File. Pass/fail is exclusively the exit
// code Dagger reports for the exec: a non-nil error here always means the
// underlying go invocation failed (or the container never ran at all),
// never a partial or inferred result.
func (t *GoCommand) Test(ctx context.Context, source *dagger.Directory) (*dagger.File, error) {
	if t.Client == nil {
		return nil, errors.New("gocommand: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("gocommand: source directory is nil")
	}
	if strings.TrimSpace(t.Command) == "" {
		return nil, errors.New("gocommand: command is required")
	}

	goVersion := resolveGoVersion(t.GoVersion)
	sourceDir := daggerkit.NewDaggerDirectoryAdapter(source)

	container := t.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/src", sourceDir).
		WithWorkdir("/src").
		WithEnvVariable("GO111MODULE", "on") // no CGO_ENABLED — Command is arbitrary, pinning either value would silently override the caller's own build settings (same rationale as gointegrationtester.go's own doc comment)
	container = mountGoCaches(t.Client, container)

	container = container.WithExec(t.goCommandArgs(), daggerkit.DaggerContainerWithExecOpts{})

	output, err := container.Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("gocommand: go command failed: %w", err)
	}

	reportContainer := container.WithNewFile("/tmp/command-output.txt", output)
	return reportContainer.File("/tmp/command-output.txt").GetRealFile(), nil
}

// goCommandArgs builds the `go <args>` invocation from t's configuration.
// See goCommandArgsFor for the tokenization/-C insertion logic, extracted
// as a pure helper so it is unit-testable without a Dagger client.
func (t *GoCommand) goCommandArgs() []string {
	return goCommandArgsFor(t.WorkDir, t.Command)
}

// goCommandArgsFor tokenizes command on whitespace (no shell, no quoting —
// see GoCommand's own doc comment) and prepends "go", inserting
// "-C workDir" immediately before the subcommand token when workDir is
// set — e.g. goCommandArgsFor("cmd/api", "build -o bin/api .") ->
// ["go", "-C", "cmd/api", "build", "-o", "bin/api", "."]. Unlike
// cargoCommandArgsFor (rustcommand.go), which inserts --manifest-path AFTER
// the subcommand, -C must be argv[1] — go itself requires -C to appear
// before the subcommand.
func goCommandArgsFor(workDir, command string) []string {
	tokens := strings.Fields(command)
	args := []string{"go"}
	if len(tokens) == 0 {
		return args
	}

	if workDir != "" {
		args = append(args, "-C", workDir)
	}
	return append(args, tokens...)
}
