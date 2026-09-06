#!/usr/bin/env bash
# Determines whether provider-path changed since its last tag-prefix tag
# and, if so, computes the next version and pushes it. Reads PROVIDER_PATH,
# TAG_PREFIX, and SOURCE_BRANCH from the environment (set by
# action.yml's composite step) and writes tagged/tag to GITHUB_OUTPUT.
#
# Extracted out of action.yml's inline `run:` block so its version-bump
# logic is unit-testable in isolation (see test.sh) without a live GitHub
# Actions runner.
set -euo pipefail

: "${PROVIDER_PATH:?PROVIDER_PATH must be set}"
: "${TAG_PREFIX:?TAG_PREFIX must be set}"
: "${SOURCE_BRANCH:?SOURCE_BRANCH must be set}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT must be set}"

LATEST_TAG=$(git tag --list "${TAG_PREFIX}*" --sort=-v:refname | head -n1)

if [ -n "$LATEST_TAG" ]; then
  RANGE="$LATEST_TAG..HEAD"
else
  RANGE="HEAD"
fi

if [ -z "$(git log --oneline "$RANGE" -- "$PROVIDER_PATH")" ]; then
  echo "⏭️  No changes under $PROVIDER_PATH since ${LATEST_TAG:-the beginning} -- skipping tag"
  echo "tagged=false" >> "$GITHUB_OUTPUT"
  echo "tag=" >> "$GITHUB_OUTPUT"
  exit 0
fi

if [ -n "$LATEST_TAG" ]; then
  VERSION_NUMBER="${LATEST_TAG#"$TAG_PREFIX"}"
else
  VERSION_NUMBER='0.0.0'
fi

# Same source-branch/commit-message heuristic as release.yml's own
# "Determine release type and version" step, but scoped to only the
# commits in $RANGE that touched $PROVIDER_PATH -- a commit that never
# touched this provider must never decide its bump, and a commit that did
# touch it must never be missed just because it isn't HEAD (design.md D6,
# per-provider blast radius).
PROVIDER_COMMIT_MSGS=$(git log --pretty=%B "$RANGE" -- "$PROVIDER_PATH")
if echo "$SOURCE_BRANCH" | grep -qiE '^hotfix/'; then
  echo "🔥 Hotfix detected from branch: $SOURCE_BRANCH -- bumping patch version"
  NEW_VERSION=$(echo "$VERSION_NUMBER" | awk -F. '{print $1"."$2"."$3+1}')
elif echo "$PROVIDER_COMMIT_MSGS" | grep -qiE '^(breaking|BREAKING)'; then
  echo "⚠️  Breaking change detected in $PROVIDER_PATH: bumping major version"
  NEW_VERSION=$(echo "$VERSION_NUMBER" | awk -F. '{print $1+1".0.0"}')
elif echo "$PROVIDER_COMMIT_MSGS" | grep -qiE '^(feat|feature)(\(.+\))?:'; then
  echo "📦 Feature detected in $PROVIDER_PATH: bumping minor version"
  NEW_VERSION=$(echo "$VERSION_NUMBER" | awk -F. '{print $1"."$2+1".0"}')
else
  echo "📝 Default for develop: bumping minor version"
  NEW_VERSION=$(echo "$VERSION_NUMBER" | awk -F. '{print $1"."$2+1".0"}')
fi

# Major is capped at 1 (design.md D6): a provider module's own /vN
# path-major-suffix rule means major >= 2 needs a deliberate module path
# migration, never an automatic bump.
NEW_MAJOR=$(echo "$NEW_VERSION" | cut -d. -f1)
if [ "$NEW_MAJOR" -ge 2 ]; then
  echo "::error::Computed major version $NEW_MAJOR for $TAG_PREFIX would exceed design.md D6's major<=1 cap -- a major bump for this provider needs a manual module-path migration, not an automatic tag."
  exit 1
fi

NEW_TAG="${TAG_PREFIX}${NEW_VERSION}"
echo "🏷️  Tagging $PROVIDER_PATH: $NEW_TAG"
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"
git push origin "$NEW_TAG"

echo "tagged=true" >> "$GITHUB_OUTPUT"
echo "tag=$NEW_TAG" >> "$GITHUB_OUTPUT"
