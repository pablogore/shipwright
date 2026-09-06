#!/usr/bin/env bash
set -euo pipefail

# Downloads the full published release asset set and verifies it exactly as
# an external consumer would (RELEASE-DIST-01B PR3 post-publish gate): the
# published set is exactly the 13 expected artifacts (6 raw binaries + 6
# archives + checksums.txt, matching .dagger/release_package.go's
# expectedPackageFiles), checksums.txt has exactly 12 entries and validates
# against every downloaded artifact, and the native linux/amd64 binary
# reports the expected version and commit when executed.
#
# This supersedes verify-release-asset.sh, which checked only a single
# asset -- a partial or corrupted publish could pass that check while still
# leaving the release incomplete.
#
# Usage: scripts/verify-release-set.sh <tag> <expected-version> <expected-commit>

TAG="${1:?tag required}"
EXPECTED_VERSION="${2:?expected version required}"
EXPECTED_COMMIT="${3:?expected commit required}"

EXPECTED_FILES=(
  shipwright-linux-amd64
  shipwright-linux-arm64
  shipwright-darwin-amd64
  shipwright-darwin-arm64
  shipwright-windows-amd64.exe
  shipwright-windows-arm64.exe
  "shipwright_${EXPECTED_VERSION}_linux_amd64.tar.gz"
  "shipwright_${EXPECTED_VERSION}_linux_arm64.tar.gz"
  "shipwright_${EXPECTED_VERSION}_darwin_amd64.tar.gz"
  "shipwright_${EXPECTED_VERSION}_darwin_arm64.tar.gz"
  "shipwright_${EXPECTED_VERSION}_windows_amd64.zip"
  "shipwright_${EXPECTED_VERSION}_windows_arm64.zip"
  checksums.txt
)

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo "==> Downloading full release asset set from ${TAG}"
gh release download "$TAG" --dir .

echo "==> Verifying exactly the expected ${#EXPECTED_FILES[@]}-artifact set is present"
ACTUAL_COUNT=$(find . -maxdepth 1 -type f | wc -l | tr -d ' ')
if [ "$ACTUAL_COUNT" -ne "${#EXPECTED_FILES[@]}" ]; then
  echo "error: downloaded ${ACTUAL_COUNT} files, want exactly ${#EXPECTED_FILES[@]}" >&2
  ls -la
  exit 1
fi
for f in "${EXPECTED_FILES[@]}"; do
  if [ ! -f "$f" ]; then
    echo "error: expected asset ${f} not found in published release" >&2
    exit 1
  fi
done

echo "==> Verifying checksums.txt has exactly 12 entries (every artifact except itself)"
CHECKSUM_ENTRIES=$(wc -l < checksums.txt | tr -d ' ')
if [ "$CHECKSUM_ENTRIES" -ne 12 ]; then
  echo "error: checksums.txt has ${CHECKSUM_ENTRIES} entries, want exactly 12" >&2
  exit 1
fi

echo "==> Verifying checksums.txt against every downloaded artifact"
sha256sum --check --strict checksums.txt

echo "==> Executing native binary"
chmod +x shipwright-linux-amd64
OUTPUT="$(./shipwright-linux-amd64 --version 2>&1)"
echo "$OUTPUT"

echo "==> Verifying version"
if ! echo "$OUTPUT" | grep -Fq "version=${EXPECTED_VERSION}"; then
  echo "error: expected version=${EXPECTED_VERSION} not found in output" >&2
  exit 1
fi

echo "==> Verifying commit"
if ! echo "$OUTPUT" | grep -Fq "git_commit=${EXPECTED_COMMIT}"; then
  echo "error: expected git_commit=${EXPECTED_COMMIT} not found in output" >&2
  exit 1
fi

echo "✅ verified full release set for ${TAG} (${#EXPECTED_FILES[@]} artifacts, checksums valid, version=${EXPECTED_VERSION}, commit=${EXPECTED_COMMIT})"
