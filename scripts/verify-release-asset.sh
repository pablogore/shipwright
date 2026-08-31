#!/usr/bin/env bash
set -euo pipefail

# Downloads a real asset back from an already-published GitHub Release and
# verifies it exactly as an external consumer would: checksum, then execute
# and confirm the reported version and commit match what was requested.
#
# Usage: scripts/verify-release-asset.sh <tag> <asset-name> <expected-version> <expected-commit>

TAG="${1:?tag required}"
ASSET_NAME="${2:?asset name required}"
EXPECTED_VERSION="${3:?expected version required}"
EXPECTED_COMMIT="${4:?expected commit required}"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo "==> Downloading ${ASSET_NAME} and checksums.txt from release ${TAG}"
gh release download "$TAG" --pattern "$ASSET_NAME" --pattern 'checksums.txt' --clobber

echo "==> Verifying checksum"
awk -v f="$ASSET_NAME" '$2==f' checksums.txt > asset.sha256
if [ ! -s asset.sha256 ]; then
  echo "error: no checksum entry found for ${ASSET_NAME} in checksums.txt" >&2
  exit 1
fi
sha256sum --check --strict asset.sha256

echo "==> Executing binary"
chmod +x "$ASSET_NAME"
OUTPUT="$(./"$ASSET_NAME" --version 2>&1)"
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

echo "✅ verified ${ASSET_NAME} from release ${TAG} (version=${EXPECTED_VERSION}, commit=${EXPECTED_COMMIT})"
