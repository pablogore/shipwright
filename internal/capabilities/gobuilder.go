// Package capabilities provides standalone, independently composable
// implementations of Shipwright's Layer 1 capability interfaces
// (pkg/shipwright). Every type in this flat package implements exactly one
// capability and carries no shared "stack" identity — design.md D-F
// explicitly rejects both a preset registry keyed by a stack name and a
// nested goservice/ subdirectory, because the path itself would be a
// bundling identity. See naming_test.go for the enforcing golden test.
//
// This package is purely additive. internal/pipelines/go-service/** is
// left in place, untouched, and still fully functional; nothing here
// migrates or removes any existing consumer of go-service — that
// migration is a later, separate work unit (design.md's Migration
// Sequence, slices 10-11).
package capabilities

import (
	"context"
	"errors"
	"fmt"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// defaultGoVersion mirrors the legacy go-service pipeline's own default
// (internal/pipelines/go-service/pipeline.go), used whenever a caller
// leaves the Go toolchain version unspecified.
const defaultGoVersion = "1.25.5"

// defaultBinaryName mirrors the legacy go-service pipeline's default
// output binary name (internal/pipelines/go-service/options.go's
// DefaultGoServiceOptions).
const defaultBinaryName = "app"

// GoBuilder builds a Go source Directory into a Directory containing the
// compiled binary. Extracted from the legacy go-service pipeline's Build /
// buildBinary logic (internal/pipelines/go-service/pipeline.go, via
// internal/pipelines/shared.GoBuilder.Build) — see design.md's "go-service
// capability mapping" table (D-F).
//
// Behavioral judgment call: the legacy pipeline's buildDocker path built a
// *dagger.Container directly from a Dockerfile via Directory.DockerBuild(),
// which cannot be expressed as this capability's Directory-typed return
// (design.md D-A's signature rule: capability interface methods use only
// Dagger core types/scalars). GoBuilder always performs the binary-compile
// path; producing and publishing a container image from the resulting
// Directory is ContainerPublisher's job (Artifactor.Publish), consistent
// with design.md's Data Flow diagram (Build -> Directory -> Artifact).
type GoBuilder struct {
	// Client is the Dagger client used to construct the build container.
	Client *dagger.Client
	// Config configures the Go toolchain version and output binary name.
	Config shipwright.BuildConfig
}

// Compile-time conformance assertion (tasks.md 3.5): GoBuilder must
// satisfy Layer 1's Builder interface (pkg/shipwright, WITH error return —
// see design.md D-A's Layer 1/Layer 2 asymmetry note).
var _ shipwright.Builder = (*GoBuilder)(nil)

// Build compiles source into a binary and returns a Directory containing
// only the compiled artifact (rooted at "/", holding the binary file).
func (b *GoBuilder) Build(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	if b.Client == nil {
		return nil, errors.New("gobuilder: dagger client is not configured")
	}
	if source == nil {
		return nil, errors.New("gobuilder: source directory is nil")
	}

	goVersion := resolveGoVersion(b.Config.GoVersion)
	binaryName := resolveBinaryName(b.Config.BinaryName)
	outPath := "/output/" + binaryName

	container := b.Client.Container().
		From("golang:"+goVersion).
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithEnvVariable("GOPATH", "/go").
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build", "-ldflags=-s -w", "-o", outPath, "."})

	built, err := container.Sync(ctx)
	if err != nil {
		return nil, fmt.Errorf("gobuilder: failed to build go binary: %w", err)
	}

	return built.Directory("/output"), nil
}

// resolveGoVersion returns cfgVersion, or defaultGoVersion when cfgVersion
// is empty. Extracted as a pure helper so it is unit-testable without a
// Dagger client.
func resolveGoVersion(cfgVersion string) string {
	if cfgVersion == "" {
		return defaultGoVersion
	}
	return cfgVersion
}

// resolveBinaryName returns cfgName, or defaultBinaryName when cfgName is
// empty. Extracted as a pure helper so it is unit-testable without a
// Dagger client.
func resolveBinaryName(cfgName string) string {
	if cfgName == "" {
		return defaultBinaryName
	}
	return cfgName
}
