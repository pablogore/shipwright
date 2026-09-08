#!/usr/bin/env bash
# Verifies a downloaded Shipwright binary against the checksums.txt
# published in the same GitHub Release (docs/RELEASE_INTEGRITY_CONTRACT.md:
# .dagger/release_package.go generates one combined checksums.txt covering
# every published artifact). Fails closed: a missing checksums file, a
# missing entry for this exact filename, or a hash mismatch all exit
# non-zero. This must run before the binary is ever executed, and must run
# on a cache hit too -- a cached copy is not exempt from verification.
#
# Usage: verify-checksum.sh <checksums-file> <binary-file> <binary-filename>
set -euo pipefail

CHECKSUMS_FILE="${1:?checksums file required}"
BINARY_FILE="${2:?binary file required}"
BINARY_NAME="${3:?binary filename required}"

if [ ! -f "$CHECKSUMS_FILE" ]; then
  echo "::error::checksums file not found at ${CHECKSUMS_FILE}. Refusing to trust an unverifiable binary." >&2
  exit 1
fi

if [ ! -f "$BINARY_FILE" ]; then
  echo "::error::binary file not found at ${BINARY_FILE}." >&2
  exit 1
fi

# checksums.txt is sha256sum's own output format: "<hash>  <filename>".
# Anchored on the filename at end-of-line so "shipwright-linux-amd64" never
# matches the "shipwright-linux-amd64.exe" or archive entries also present
# in the same combined file.
EXPECTED_LINE="$(grep -E "[[:space:]]\*?${BINARY_NAME}\$" "$CHECKSUMS_FILE" || true)"
if [ -z "$EXPECTED_LINE" ]; then
  echo "::error::No checksum entry for '${BINARY_NAME}' in checksums.txt. Refusing to trust an unverifiable binary." >&2
  exit 1
fi
EXPECTED_HASH="$(awk '{print $1}' <<<"$EXPECTED_LINE")"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL_HASH="$(sha256sum "$BINARY_FILE" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL_HASH="$(shasum -a 256 "$BINARY_FILE" | awk '{print $1}')"
else
  echo "::error::Neither sha256sum nor shasum is available to verify the binary." >&2
  exit 1
fi

if [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
  echo "::error::Checksum mismatch for ${BINARY_NAME}: expected ${EXPECTED_HASH}, got ${ACTUAL_HASH}. Refusing to execute an unverified binary." >&2
  exit 1
fi

echo "✅ checksum verified for ${BINARY_NAME}"
