// RELEASE-DIST-01B PR2: packaging + checksums.
//
// Builds on ReleaseBuild/ReleaseVerify (release.go, PR1): takes the 6 raw
// platform binaries and produces exactly the distributable set GitHub
// Release expects today (docs/RELEASE_INTEGRITY_CONTRACT.md) -- the same 6
// raw binaries, one archive per platform (tar.gz for linux/darwin, zip for
// windows, matching .goreleaser.yml's archive contents and name_template),
// and a single combined checksums.txt covering all 12 artifacts. Publishing
// (uploading any of this to a GitHub Release) is PR3's job, not this one.
package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/shipwright/internal/dagger"
)

// packagingImage is the container used to build archives and compute
// checksums. It needs tar (present via busybox on alpine) and zip (not
// present by default, installed explicitly) plus sha256sum for checksums.
const packagingImage = "alpine:3.22"

// archiveExt returns the archive format GoReleaser uses for a platform:
// .zip on Windows, .tar.gz everywhere else (.goreleaser.yml's
// format_overrides).
func (p releasePlatform) archiveExt() string {
	if p.os == "windows" {
		return "zip"
	}
	return "tar.gz"
}

// archiveName returns the release archive name for a platform, matching
// .goreleaser.yml's name_template: shipwright_<version>_<os>_<arch>.<ext>.
// os/arch are the raw (lowercase) GOOS/GOARCH values -- release.yml's own
// "Prepare raw binaries" step globs shipwright_*_linux_amd64.tar.gz etc.,
// confirming GoReleaser does not title-case these.
func (p releasePlatform) archiveName(version string) string {
	return fmt.Sprintf("shipwright_%s_%s_%s.%s", version, p.os, p.arch, p.archiveExt())
}

// packagedFileName is the binary's name *inside* an archive: plain
// "shipwright" (or "shipwright.exe" on Windows), never the platform-suffixed
// release asset name -- matching release.yml's extraction step, which pulls
// a file literally named "shipwright"/"shipwright.exe" out of each archive.
func (p releasePlatform) packagedFileName() string {
	if p.os == "windows" {
		return "shipwright.exe"
	}
	return "shipwright"
}

// archiveFor builds the release archive for one platform: the binary
// (renamed to packagedFileName) plus README.md, LICENSE, and CHANGELOG.md
// from source, matching .goreleaser.yml's archives.files list.
func archiveFor(source *dagger.Directory, p releasePlatform, binary *dagger.File, version string) *dagger.File {
	name := p.archiveName(version)
	staged := dag.Container().
		From(packagingImage).
		WithExec([]string{"apk", "add", "--no-cache", "zip"}).
		WithMountedFile("/stage/"+p.packagedFileName(), binary).
		WithMountedFile("/stage/README.md", source.File("README.md")).
		WithMountedFile("/stage/LICENSE", source.File("LICENSE")).
		WithMountedFile("/stage/CHANGELOG.md", source.File("CHANGELOG.md")).
		WithWorkdir("/stage").
		WithExec([]string{"mkdir", "-p", "/out"})

	if p.os == "windows" {
		staged = staged.WithExec([]string{"zip", "-q", "/out/" + name, p.packagedFileName(), "README.md", "LICENSE", "CHANGELOG.md"})
	} else {
		staged = staged.WithExec([]string{"tar", "-czf", "/out/" + name, p.packagedFileName(), "README.md", "LICENSE", "CHANGELOG.md"})
	}

	return staged.File("/out/" + name)
}

// ReleasePackage runs ReleaseBuild and packages its output into the full
// distributable artifact set: the 6 raw binaries (unchanged), one archive
// per platform, and a combined checksums.txt covering all 12 files -- the
// same shape release.yml's "Prepare raw binaries" + "Merge raw-binary
// checksums" steps produce today, just built by Dagger instead of shell
// glob loops. Publishing is out of scope here (PR3).
func (m *Shipwright) ReleasePackage(ctx context.Context, source *dagger.Directory, version, commit, buildTime string) (*dagger.Directory, error) {
	build, err := m.ReleaseBuild(ctx, source, version, commit, buildTime)
	if err != nil {
		return nil, err
	}

	output := dag.Directory()
	for _, p := range releaseMatrix {
		binary := build.File(p.binaryName())
		output = output.WithFile(p.binaryName(), binary)
		output = output.WithFile(p.archiveName(version), archiveFor(source, p, binary, version))
	}

	if _, err := output.Sync(ctx); err != nil {
		return nil, fmt.Errorf("release packaging failed: %w", err)
	}

	// checksums.txt is generated from expectedPackageFiles (minus its own
	// trailing "checksums.txt" entry, which does not exist yet), the same
	// explicit, deterministically-ordered list ReleasePackageVerify checks
	// against -- not from a shell glob, so generation and verification always
	// agree on exactly which files belong in the set.
	expected := expectedPackageFiles(version)
	artifactFiles := expected[:len(expected)-1]
	checksumArgs := append([]string{"sha256sum", "--"}, artifactFiles...)
	checksummed := dag.Container().
		From(packagingImage).
		WithMountedDirectory("/artifacts", output).
		WithWorkdir("/artifacts").
		WithExec([]string{"sh", "-c", strings.Join(checksumArgs, " ") + " > checksums.txt"}).
		Directory("/artifacts")

	if _, err := checksummed.Sync(ctx); err != nil {
		return nil, fmt.Errorf("checksum generation failed: %w", err)
	}

	return checksummed, nil
}

// expectedPackageFiles lists every file ReleasePackage must produce for a
// given version: 6 raw binaries + 6 archives + checksums.txt.
func expectedPackageFiles(version string) []string {
	files := make([]string, 0, len(releaseMatrix)*2+1)
	for _, p := range releaseMatrix {
		files = append(files, p.binaryName(), p.archiveName(version))
	}
	return append(files, "checksums.txt")
}

// ReleasePackageVerify runs ReleasePackage and validates the SHA256
// checksums contract without publishing anything: every expected raw
// binary, archive, and checksums.txt itself must be present, and
// checksums.txt must correctly describe every artifact it lists (an
// integrity guarantee over this generated set -- not a reproducibility
// guarantee, since tar/zip can embed filesystem timestamps that vary
// between runs) -- fails closed on a missing artifact, an extra/missing
// checksum entry, or any hash mismatch.
func (m *Shipwright) ReleasePackageVerify(ctx context.Context, source *dagger.Directory, version, commit, buildTime string) (string, error) {
	pkg, err := m.ReleasePackage(ctx, source, version, commit, buildTime)
	if err != nil {
		return "", err
	}

	entries, err := pkg.Entries(ctx)
	if err != nil {
		return "", fmt.Errorf("could not list release package output: %w", err)
	}
	present := make(map[string]bool, len(entries))
	for _, e := range entries {
		present[e] = true
	}

	expected := expectedPackageFiles(version)
	var missing []string
	for _, name := range expected {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("release package is incomplete, missing: %s", strings.Join(missing, ", "))
	}
	if len(entries) != len(expected) {
		return "", fmt.Errorf("release package has %d files, want exactly %d (expected: %s)", len(entries), len(expected), strings.Join(expected, ", "))
	}

	checkOutput, err := dag.Container().
		From(packagingImage).
		WithMountedDirectory("/artifacts", pkg).
		WithWorkdir("/artifacts").
		WithExec([]string{"sh", "-c", "sha256sum -c checksums.txt"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("checksum verification failed: %w", err)
	}

	okCount := strings.Count(checkOutput, ": OK")
	if okCount != len(expected)-1 { // checksums.txt does not check itself
		return "", fmt.Errorf("expected %d verified checksums, got %d: %s", len(expected)-1, okCount, checkOutput)
	}

	return fmt.Sprintf("release package verified: %d artifacts present, %d checksums confirmed", len(expected), okCount), nil
}
