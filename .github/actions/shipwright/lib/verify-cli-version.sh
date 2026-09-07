#!/usr/bin/env bash
# Confirms the downloaded binary actually reports the version it was
# supposed to be. Reuses the exact stable contract scripts/verify-release-set.sh
# already relies on post-publish: `shipwright --version`'s combined
# stdout+stderr contains a literal `version=<X.Y.Z>` substring, with no
# leading "v" (main.go's showVersion logs the raw ldflags-injected Version,
# and release.yml passes that ldflags value without a "v" prefix -- see
# .dagger/release.go's releaseLdflags and release.yml's `VERSION=$(echo
# "$TAG" | sed 's/^v//')`). Grepping for that exact substring is far more
# robust than parsing positional fields out of human-readable log
# formatting, which can reorder or reword without notice.
#
# Usage: verify-cli-version.sh <binary-path> <expected-version-tag>
set -euo pipefail

BINARY_PATH="${1:?binary path required}"
EXPECTED_TAG="${2:?expected version tag required}"
EXPECTED_VERSION="${EXPECTED_TAG#v}"

OUTPUT="$("$BINARY_PATH" --version 2>&1)" || {
  echo "::error::'${BINARY_PATH} --version' failed to run:" >&2
  echo "$OUTPUT" >&2
  exit 1
}

if ! grep -Fq "version=${EXPECTED_VERSION}" <<<"$OUTPUT"; then
  echo "::error::Downloaded binary does not report the expected version. Wanted 'version=${EXPECTED_VERSION}', got:" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

echo "$OUTPUT"
