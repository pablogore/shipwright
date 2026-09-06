#!/bin/bash

# changelog-prepend-release.sh - Insert a generated release entry into
# CHANGELOG.md ahead of its existing content, re-opening a fresh
# [Unreleased] section afterward (RELEASE-DIST-01A).
#
# Idempotent by tag: if CHANGELOG.md already contains the "## Release
# <tag>" header this script's caller writes, it is a no-op. This is what
# makes a retried release run safe -- it will not duplicate the entry.
#
# Usage: ./scripts/changelog-prepend-release.sh <changelog-file> <new-entry-file> <tag>
# Rewrites <changelog-file> in place. <new-entry-file> must start with a
# "## Release <tag>" line (this is what the release workflow's changelog
# generator writes).

set -euo pipefail

CHANGELOG_FILE="${1:?changelog file required}"
NEW_ENTRY_FILE="${2:?new entry file required}"
TAG="${3:?tag required}"

if [ -f "$CHANGELOG_FILE" ] && grep -qxF "## Release ${TAG}" "$CHANGELOG_FILE"; then
  echo "${CHANGELOG_FILE} already contains an entry for ${TAG} -- skipping (idempotent)."
  exit 0
fi

if [ ! -f "$CHANGELOG_FILE" ]; then
  cp "$NEW_ENTRY_FILE" "$CHANGELOG_FILE"
  exit 0
fi

TMP_TAIL="$(mktemp)"
TMP_OUT="$(mktemp)"
trap 'rm -f "$TMP_TAIL" "$TMP_OUT"' EXIT

# Drop the [Unreleased] heading and its body, stopping at the NEXT release
# heading -- either a bracketed historical one ("## [x.y.z] - date") or one
# this very script generates ("## Release <tag>") -- which is kept, along
# with everything after it. Both forms must be recognized: if a human ever
# restores [Unreleased] to the top of the file (the normal Keep a Changelog
# layout), the next generated "## Release <tag>" entry sits directly below
# it, and a stop-condition that only matched "## [" would swallow that entry
# into the deleted region. A plain `/^## \[Unreleased\]/,$d` range deletes to
# end-of-file instead -- that would erase every historical release section
# on each run.
awk '
  /^## \[Unreleased\]/ { skipping = 1; next }
  skipping && /^## (\[|Release )/ { skipping = 0 }
  !skipping { print }
' "$CHANGELOG_FILE" > "$TMP_TAIL"

{
  cat "$NEW_ENTRY_FILE"
  echo ""
  echo "---"
  echo ""
  cat "$TMP_TAIL"
} > "$TMP_OUT"

if grep -q "^## \[Unreleased\]" "$CHANGELOG_FILE"; then
  {
    cat "$TMP_OUT"
    echo ""
    echo "## [Unreleased]"
    echo ""
    echo "### Added"
    echo "- Changes in progress..."
  } > "$CHANGELOG_FILE"
else
  mv "$TMP_OUT" "$CHANGELOG_FILE"
fi
