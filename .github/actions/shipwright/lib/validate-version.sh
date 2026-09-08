#!/usr/bin/env bash
# Enforces RELEASE-DIST-01C's pinned-only distribution model: version must
# be an explicit tag, never empty and never the literal "latest" -- there is
# no dynamic "resolve the newest release" path in this action, so a new
# Shipwright release can never silently change what a workflow runs.
#
# Usage: validate-version.sh <version>
# On success, appends version=<value> to $GITHUB_OUTPUT.
set -euo pipefail

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
  echo "::error::The 'version' input is required and must be an explicit release tag (e.g. v1.2.3). There is no default -- pin the version you depend on." >&2
  exit 1
fi

if [ "$VERSION" = "latest" ]; then
  echo "::error::version=latest is not supported. Pin an explicit vX.Y.Z tag instead." >&2
  exit 1
fi

echo "version=${VERSION}" >> "$GITHUB_OUTPUT"
