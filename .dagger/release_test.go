package main

import (
	"context"
	"strings"
	"testing"
)

// TestReleaseMatrix_SixPlatforms locks the matrix to RELEASE-DIST-01B's
// explicit scope (linux/darwin/windows x amd64/arm64) so an accidental
// addition or removal is caught here rather than discovered at release
// time.
func TestReleaseMatrix_SixPlatforms(t *testing.T) {
	if len(releaseMatrix) != 6 {
		t.Fatalf("releaseMatrix has %d platforms, want 6", len(releaseMatrix))
	}

	want := map[string]bool{
		"linux/amd64": true, "linux/arm64": true,
		"darwin/amd64": true, "darwin/arm64": true,
		"windows/amd64": true, "windows/arm64": true,
	}
	for _, p := range releaseMatrix {
		key := p.os + "/" + p.arch
		if !want[key] {
			t.Errorf("releaseMatrix contains unexpected platform %s", key)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Errorf("releaseMatrix is missing platforms: %v", want)
	}
}

// TestReleasePlatform_BinaryName pins the asset-naming contract documented
// in docs/RELEASE_INTEGRITY_CONTRACT.md: shipwright-<os>-<arch>, with a
// .exe suffix on Windows only.
func TestReleasePlatform_BinaryName(t *testing.T) {
	cases := []struct {
		p    releasePlatform
		want string
	}{
		{releasePlatform{os: "linux", arch: "amd64"}, "shipwright-linux-amd64"},
		{releasePlatform{os: "linux", arch: "arm64"}, "shipwright-linux-arm64"},
		{releasePlatform{os: "darwin", arch: "amd64"}, "shipwright-darwin-amd64"},
		{releasePlatform{os: "darwin", arch: "arm64"}, "shipwright-darwin-arm64"},
		{releasePlatform{os: "windows", arch: "amd64"}, "shipwright-windows-amd64.exe"},
		{releasePlatform{os: "windows", arch: "arm64"}, "shipwright-windows-arm64.exe"},
	}
	for _, c := range cases {
		if got := c.p.binaryName(); got != c.want {
			t.Errorf("binaryName() for %s/%s = %q, want %q", c.p.os, c.p.arch, got, c.want)
		}
	}
}

// TestReleaseLdflags_InjectsAllThreeFields asserts the -X injection string
// carries all three fields (formerly matching .goreleaser.yml's ldflags
// template, retired in PR3), so a binary built here reports the expected
// version metadata for its tag/commit.
func TestReleaseLdflags_InjectsAllThreeFields(t *testing.T) {
	got := releaseLdflags("v1.2.3", "abc1234", "2026-01-01T00:00:00Z")

	for _, want := range []string{
		"-X main.Version=v1.2.3",
		"-X main.GitCommit=abc1234",
		"-X main.BuildTime=2026-01-01T00:00:00Z",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("releaseLdflags() = %q, want it to contain %q", got, want)
		}
	}
}

// TestResolveGoVersion exercises the .go-version fail-closed contract
// (RELEASE-DIST-01B PR3): a live Dagger session is required to build a
// source Directory, so this only runs under `dagger run go test` (see
// `make dagger-test`), same as every other test in this package.
func TestResolveGoVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("valid version is trimmed", func(t *testing.T) {
		src := dag.Directory().WithNewFile(".go-version", "1.26.7\n")
		got, err := resolveGoVersion(ctx, src)
		if err != nil {
			t.Fatalf("resolveGoVersion() error = %v, want nil", err)
		}
		if got != "1.26.7" {
			t.Errorf("resolveGoVersion() = %q, want %q", got, "1.26.7")
		}
	})

	t.Run("whitespace-only file fails closed", func(t *testing.T) {
		src := dag.Directory().WithNewFile(".go-version", "   \n")
		if _, err := resolveGoVersion(ctx, src); err == nil {
			t.Fatal("resolveGoVersion() = nil error, want an error for an empty .go-version")
		}
	})

	t.Run("missing file fails closed", func(t *testing.T) {
		src := dag.Directory()
		if _, err := resolveGoVersion(ctx, src); err == nil {
			t.Fatal("resolveGoVersion() = nil error, want an error for a missing .go-version")
		}
	})
}

// TestReleaseBuild_RejectsEmptyInputs is a pure control-flow test: the
// version/commit/buildTime validation must reject empty (or whitespace-only)
// input before any container is created, matching the "fail closed" shape
// used throughout this module (see capabilities.go's Plan.Execute).
func TestReleaseBuild_RejectsEmptyInputs(t *testing.T) {
	m := &Shipwright{}
	ctx := context.Background()

	cases := []struct {
		name                       string
		version, commit, buildTime string
	}{
		{"empty version", "", "abc1234", "2026-01-01T00:00:00Z"},
		{"whitespace version", "   ", "abc1234", "2026-01-01T00:00:00Z"},
		{"empty commit", "v1.0.0", "", "2026-01-01T00:00:00Z"},
		{"empty buildTime", "v1.0.0", "abc1234", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.ReleaseBuild(ctx, nil, c.version, c.commit, c.buildTime)
			if err == nil {
				t.Fatalf("ReleaseBuild(%q, %q, %q) = nil error, want an error", c.version, c.commit, c.buildTime)
			}
		})
	}
}
