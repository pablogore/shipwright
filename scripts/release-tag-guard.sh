#!/bin/bash

# release-tag-guard.sh - Fail-closed create-or-verify for an immutable
# release tag (RELEASE-DIST-01A).
#
# Given a tag name and the current commit:
#   - tag doesn't exist                -> create it, push it, exit 0
#   - tag exists at the current commit -> no-op (safe same-commit retry), exit 0
#   - tag exists at a different commit -> FAIL CLOSED, exit 1
#
# A release tag is immutable once published: this script will never delete,
# move, or force-push over an existing tag. If a tag was published pointing
# at the wrong commit, that must be resolved by a human, never silently by
# this workflow.
#
# Usage: ./scripts/release-tag-guard.sh <tag>
# Must run inside a git checkout with `origin` reachable and tags fetched
# (fetch-depth: 0, fetch-tags: true).

set -euo pipefail

TAG="${1:?tag required}"
CURRENT_COMMIT="$(git rev-parse HEAD)"

if git tag -l | grep -qx "$TAG"; then
  TAG_COMMIT="$(git rev-parse "${TAG}^{commit}" 2>/dev/null || echo "")"

  if [ "$TAG_COMMIT" != "$CURRENT_COMMIT" ]; then
    echo "::error::Tag ${TAG} already exists and points to commit ${TAG_COMMIT}, but this release run targets commit ${CURRENT_COMMIT}." >&2
    echo "::error::Refusing to move an existing release tag -- a release tag is immutable once published." >&2
    echo "::error::If ${TAG} was published in error, resolve it manually (delete + re-tag) and re-run -- this script will never do it for you." >&2
    exit 1
  fi

  echo "Tag ${TAG} already exists and points to the current commit -- safe retry, no changes needed."
  exit 0
fi

git tag -a "$TAG" -m "Release ${TAG}"
git push origin "$TAG"
echo "Tag created and pushed: ${TAG}"
