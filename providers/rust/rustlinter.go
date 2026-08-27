package rust

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// RustLinter runs cargo clippy against a source Directory and returns its
// output as the report File. Structural mirror of providers/go's GoLinter —
// one of three independent Tester implementations, none privileged, for
// the Rust toolchain.
type RustLinter struct {
	// Client is the Dagger client used to construct the lint container.
	Client *dagger.Client
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

	container := l.Client.Container().
		From("rust:"+rustVersion).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithExec([]string{"rustup", "component", "add", "clippy"})

	output, err := container.
		WithExec([]string{"cargo", "clippy", "--all-targets", "--", "-D", "warnings"}).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("rustlinter: cargo clippy found issues: %w", err)
	}

	reportContainer := container.WithNewFile("/tmp/lint-report.txt", output)
	return reportContainer.File("/tmp/lint-report.txt"), nil
}
