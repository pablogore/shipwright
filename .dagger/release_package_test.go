package main

import (
	"strings"
	"testing"
)

// TestReleasePlatform_ArchiveExt pins the archive format (formerly
// .goreleaser.yml's format_overrides, retired in PR3): zip on Windows,
// tar.gz everywhere else.
func TestReleasePlatform_ArchiveExt(t *testing.T) {
	cases := []struct {
		p    releasePlatform
		want string
	}{
		{releasePlatform{os: "linux", arch: "amd64"}, "tar.gz"},
		{releasePlatform{os: "darwin", arch: "arm64"}, "tar.gz"},
		{releasePlatform{os: "windows", arch: "amd64"}, "zip"},
		{releasePlatform{os: "windows", arch: "arm64"}, "zip"},
	}
	for _, c := range cases {
		if got := c.p.archiveExt(); got != c.want {
			t.Errorf("archiveExt() for %s/%s = %q, want %q", c.p.os, c.p.arch, got, c.want)
		}
	}
}

// TestReleasePlatform_ArchiveName pins the release archive naming (formerly
// .goreleaser.yml's name_template, retired in PR3):
// shipwright_<version>_<os>_<arch>.<ext>, lowercase os/arch -- the exact
// shape docs/RELEASE_INTEGRITY_CONTRACT.md documents as the distribution
// contract.
func TestReleasePlatform_ArchiveName(t *testing.T) {
	cases := []struct {
		p    releasePlatform
		want string
	}{
		{releasePlatform{os: "linux", arch: "amd64"}, "shipwright_v1.2.3_linux_amd64.tar.gz"},
		{releasePlatform{os: "darwin", arch: "arm64"}, "shipwright_v1.2.3_darwin_arm64.tar.gz"},
		{releasePlatform{os: "windows", arch: "amd64"}, "shipwright_v1.2.3_windows_amd64.zip"},
	}
	for _, c := range cases {
		if got := c.p.archiveName("v1.2.3"); got != c.want {
			t.Errorf("archiveName() for %s/%s = %q, want %q", c.p.os, c.p.arch, got, c.want)
		}
	}
}

// TestReleasePlatform_PackagedFileName pins the in-archive binary name:
// plain "shipwright"/"shipwright.exe", never the platform-suffixed release
// asset name -- matching release.yml's extraction step, which pulls a file
// literally named "shipwright"/"shipwright.exe" out of each archive.
func TestReleasePlatform_PackagedFileName(t *testing.T) {
	cases := []struct {
		p    releasePlatform
		want string
	}{
		{releasePlatform{os: "linux", arch: "amd64"}, "shipwright"},
		{releasePlatform{os: "darwin", arch: "arm64"}, "shipwright"},
		{releasePlatform{os: "windows", arch: "amd64"}, "shipwright.exe"},
	}
	for _, c := range cases {
		if got := c.p.packagedFileName(); got != c.want {
			t.Errorf("packagedFileName() for %s/%s = %q, want %q", c.p.os, c.p.arch, got, c.want)
		}
	}
}

// TestExpectedPackageFiles_CountAndShape asserts the full expected-artifact
// list is exactly 13 files (6 raw binaries + 6 archives + checksums.txt),
// with no duplicates -- this is the list ReleasePackageVerify's
// completeness check is built on, so an error here would silently weaken
// that fail-closed guarantee.
func TestExpectedPackageFiles_CountAndShape(t *testing.T) {
	files := expectedPackageFiles("v1.2.3")

	if len(files) != 13 {
		t.Fatalf("expectedPackageFiles() has %d entries, want 13 (6 binaries + 6 archives + checksums.txt)", len(files))
	}

	seen := make(map[string]bool, len(files))
	for _, f := range files {
		if seen[f] {
			t.Errorf("expectedPackageFiles() contains duplicate entry %q", f)
		}
		seen[f] = true
	}

	if !seen["checksums.txt"] {
		t.Error("expectedPackageFiles() does not contain checksums.txt")
	}

	var archiveCount, binaryCount int
	for _, f := range files {
		switch {
		case strings.HasSuffix(f, ".tar.gz"), strings.HasSuffix(f, ".zip"):
			archiveCount++
		case strings.HasPrefix(f, "shipwright-"):
			binaryCount++
		}
	}
	if archiveCount != 6 {
		t.Errorf("expectedPackageFiles() has %d archives, want 6", archiveCount)
	}
	if binaryCount != 6 {
		t.Errorf("expectedPackageFiles() has %d raw binaries, want 6", binaryCount)
	}
}
