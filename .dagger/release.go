// RELEASE-DIST-01B PR1: Dagger release build contract.
//
// This file gives Dagger ownership of shipwright's release build matrix,
// independent of the generic Builder/Tester/Plan capability contract in
// capabilities.go. That contract is shipwright's public pipeline DSL for
// pipelines *users* define (single Directory in, single Directory out); the
// release build matrix below has a different shape entirely (one source in,
// N platform-specific binaries out, with version/commit metadata baked into
// each), so it is a standalone Dagger Function rather than a Builder
// implementation.
//
// Scope (PR1 only): build matrix + metadata injection + verify-only mode.
// Packaging/checksums are PR2. release.yml/GoReleaser integration is PR3.
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger/shipwright/internal/dagger"
)

// releaseGoVersion pins the toolchain used to cross-compile the release
// matrix. Kept in sync with GO_VERSION in .github/workflows/ci.yml by hand
// (this module cannot import root go.mod's version constant -- see
// capabilities.go's package doc on the Layer 1/Layer 2 module boundary).
const releaseGoVersion = "1.26.7"

// releasePlatform is one target in shipwright's release build matrix.
type releasePlatform struct {
	os   string
	arch string
}

// releaseMatrix mirrors .goreleaser.yml's builds.goos/builds.goarch matrix.
// Any change here must be made in both places until PR3 retires GoReleaser.
var releaseMatrix = []releasePlatform{
	{os: "linux", arch: "amd64"},
	{os: "linux", arch: "arm64"},
	{os: "darwin", arch: "amd64"},
	{os: "darwin", arch: "arm64"},
	{os: "windows", arch: "amd64"},
	{os: "windows", arch: "arm64"},
}

// binaryName returns the release asset name for a platform, matching the
// naming already documented in docs/RELEASE_INTEGRITY_CONTRACT.md
// (shipwright-<os>-<arch>, with a .exe suffix on Windows).
func (p releasePlatform) binaryName() string {
	name := fmt.Sprintf("shipwright-%s-%s", p.os, p.arch)
	if p.os == "windows" {
		name += ".exe"
	}
	return name
}

// ldflags builds the -X main.Version/-X main.GitCommit/-X main.BuildTime
// injection string, matching .goreleaser.yml's ldflags template exactly so
// a binary produced here and one produced by GoReleaser report identical
// version metadata for the same tag/commit.
func releaseLdflags(version, commit, buildTime string) string {
	return fmt.Sprintf("-s -w -X main.Version=%s -X main.GitCommit=%s -X main.BuildTime=%s", version, commit, buildTime)
}

// buildOne cross-compiles source for a single platform and returns the
// resulting binary as a File named per binaryName().
func (m *Shipwright) buildOne(source *dagger.Directory, p releasePlatform, version, commit, buildTime string) *dagger.File {
	outPath := "/out/" + p.binaryName()

	ctr := dag.Container().
		From("golang:"+releaseGoVersion).
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", p.os).
		WithEnvVariable("GOARCH", p.arch).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{
			"go", "build",
			"-trimpath",
			"-ldflags", releaseLdflags(version, commit, buildTime),
			"-o", outPath,
			".",
		})

	return ctr.File(outPath)
}

// ReleaseBuild cross-compiles shipwright's root binary for every platform in
// releaseMatrix, injecting version/commit/build-time metadata via ldflags,
// and returns a Directory containing one binary per platform. It never
// publishes or packages anything: given a valid version and commit, this
// function's only job is to produce exactly the raw binaries GitHub Release
// today expects (docs/RELEASE_INTEGRITY_CONTRACT.md), nothing more.
func (m *Shipwright) ReleaseBuild(ctx context.Context, source *dagger.Directory, version, commit, buildTime string) (*dagger.Directory, error) {
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("release build requires a non-empty version")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, errors.New("release build requires a non-empty commit")
	}
	if strings.TrimSpace(buildTime) == "" {
		return nil, errors.New("release build requires a non-empty buildTime")
	}

	output := dag.Directory()
	for _, p := range releaseMatrix {
		bin := m.buildOne(source, p, version, commit, buildTime)
		output = output.WithFile(p.binaryName(), bin)
	}

	if _, err := output.Sync(ctx); err != nil {
		return nil, fmt.Errorf("release build matrix failed: %w", err)
	}

	return output, nil
}

// ReleaseVerify runs the same build matrix as ReleaseBuild and additionally
// checks that it satisfies the release contract, without publishing
// anything -- the "verify-only" mode required by RELEASE-DIST-01B's
// acceptance criteria. It fails closed:
//   - any platform in releaseMatrix missing from the build output aborts
//     verification (a partial matrix is never treated as success);
//   - the linux/amd64 binary (the one platform this Dagger container can
//     actually execute) is run with --version and its output must contain
//     both the expected version and the expected commit.
//
// Cross-compiled binaries for other platforms cannot be executed inside
// this container, so their check is limited to presence and non-zero size;
// full runtime verification for those stays a release-time human/CI concern
// until a matching execution environment is wired in.
func (m *Shipwright) ReleaseVerify(ctx context.Context, source *dagger.Directory, version, commit, buildTime string) (string, error) {
	build, err := m.ReleaseBuild(ctx, source, version, commit, buildTime)
	if err != nil {
		return "", err
	}

	entries, err := build.Entries(ctx)
	if err != nil {
		return "", fmt.Errorf("could not list release build output: %w", err)
	}

	present := make(map[string]bool, len(entries))
	for _, e := range entries {
		present[e] = true
	}

	var missing []string
	for _, p := range releaseMatrix {
		if !present[p.binaryName()] {
			missing = append(missing, p.binaryName())
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("release build matrix is incomplete, missing: %s", strings.Join(missing, ", "))
	}

	for _, p := range releaseMatrix {
		size, err := build.File(p.binaryName()).Size(ctx)
		if err != nil {
			return "", fmt.Errorf("could not stat %s: %w", p.binaryName(), err)
		}
		if size == 0 {
			return "", fmt.Errorf("%s is empty", p.binaryName())
		}
	}

	native := releasePlatform{os: "linux", arch: "amd64"}
	versionOutput, err := dag.Container().
		From("alpine:3.22").
		WithFile("/usr/local/bin/shipwright", build.File(native.binaryName())).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/shipwright"}).
		WithExec([]string{"/usr/local/bin/shipwright", "--version"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("could not execute %s --version: %w", native.binaryName(), err)
	}

	if !strings.Contains(versionOutput, version) {
		return "", fmt.Errorf("%s --version output does not contain expected version %q: %s", native.binaryName(), version, versionOutput)
	}
	if !strings.Contains(versionOutput, commit) {
		return "", fmt.Errorf("%s --version output does not contain expected commit %q: %s", native.binaryName(), commit, versionOutput)
	}

	return fmt.Sprintf("release build verified: %d platforms present, %s reports version=%s commit=%s", len(releaseMatrix), native.binaryName(), version, commit), nil
}
