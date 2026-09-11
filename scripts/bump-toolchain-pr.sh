#!/bin/bash

# bump-toolchain-pr.sh - Deliver a completed toolchain version bump
# (internal/toolbump, design.md D-1/D-3/D-4) as a reviewable PR, never a
# direct push. Toolchain-agnostic successor to
# scripts/bump-go-version-pr.sh#280: branch, commit, and PR text now derive
# from `go run ./scripts/toolbump -toolchain=<id> -target=<target>
# -print-delivery`'s manifest instead of a literal "go", and the root guard
# relaxes from go.work+.dagger (Go-specific) to a single toolchain-agnostic
# `[ -e .git ]` check -- per-toolchain root markers are enforced inside the
# Go binary via Toolchain.RootMarkers(), not here (design.md's
# "Toolchain-agnostic root guard" requirement).
#
# Invoked as the LAST step of `make bump-toolchain`, after
# `go run ./scripts/toolbump -toolchain=$(TOOLCHAIN) -target=$(TARGET)` has
# already mutated every registered site and `make ci-final` has already
# validated the mutated tree in place. This script's only job is git/gh
# delivery of that already-validated diff -- it never mutates a source file
# itself.
#
# Mirrors this repo's existing precedent for multi-step git/gh automation
# (scripts/changelog-prepend-release.sh, release.yml:597-643): a namespaced
# throwaway branch, an explicit push refspec (never develop/main directly,
# both are PR-protected), and an idempotent `gh pr list` check before
# `gh pr create` so a retried run never opens a duplicate PR.
#
# Usage: ./scripts/bump-toolchain-pr.sh <toolchain-id> <target-version>
# Must run from the repository root, with the target's mutations already
# applied and validated in the working tree.

set -euo pipefail

TOOLCHAIN_ID="${1:?toolchain id required}"
TARGET="${2:?target version required}"

# --- Argument format guard (Threat Matrix: push state). Both values feed
# the explicit push refspec (HEAD:refs/heads/chore/<id>-<target>) and the
# `go run` invocation below; refusing a metacharacter in either one here,
# before any git/gh/go command runs, means neither can ever reach the
# refspec or the manifest read as anything but an already-rejected input --
# the same "fail before touching anything" guarantee
# internal/toolbump/core's own E6 (target) and toolchain id checks give a
# real bump run.
if ! [[ "$TOOLCHAIN_ID" =~ ^[a-z0-9]+$ ]]; then
  echo "::error::toolchain id must match ^[a-z0-9]+\$, got: ${TOOLCHAIN_ID}" >&2
  exit 1
fi
if ! [[ "$TARGET" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "::error::target version must match ^[0-9]+.[0-9]+.[0-9]+\$, got: ${TARGET}" >&2
  exit 1
fi

# --- Repo-root guard (Threat Matrix: git repository selection). Relaxed
# from scripts/bump-go-version-pr.sh#280's Go-specific go.work+.dagger check
# to a single toolchain-agnostic `[ -e .git ]` test -- `-e`, not `-d`,
# because a worktree's `.git` is a FILE pointing at the real git dir, not a
# directory (design.md's "Wrong directory rejected" scenario covers both
# shapes). Toolchain-specific root markers (e.g. Go's go.work/.dagger) are
# still enforced, but inside the Go binary via Toolchain.RootMarkers(), not
# here.
if [ ! -e ".git" ]; then
  echo "::error::must be run from the repository root (.git not found in $(pwd))" >&2
  exit 1
fi

# --- Clean-worktree precondition (Threat Matrix: commit state). A correct
# toolbump run only ever MODIFIES pre-existing, registered files (a
# toolchain's own Sites registry entries plus whatever its RuntimeUpgrader
# mutates) -- it never creates a new file. So the presence of ANY untracked
# ("??") entry means something unexpected happened, and this script refuses
# rather than silently discarding it or -- far worse -- sweeping it into the
# commit with a broad `git add -A`. This is also why the commit below is
# staged one known-modified path at a time instead of `-A`/`-a`: unexpected
# content is refused up front, and everything that IS staged is staged by
# explicit, individually-named path.
UNTRACKED="$(git status --porcelain | grep '^??' || true)"
if [ -n "$UNTRACKED" ]; then
  echo "::error::worktree has untracked files -- refusing to commit an unexpected diff:" >&2
  echo "$UNTRACKED" >&2
  exit 1
fi

# --- Scoped git add: every tracked file the bump modified, one path at a
# time. Never `git add -A` (would also sweep up untracked files, already
# refused above, if this check were ever weakened) and never
# `git commit -a` (implicit, un-auditable scope).
CHANGED_FILES="$(git diff --name-only)"
if [ -z "$CHANGED_FILES" ]; then
  echo "Nothing changed for ${TOOLCHAIN_ID} target ${TARGET} -- already at that version, nothing to commit."
  exit 0
fi

# --- Delivery manifest (Threat Matrix: PR commands / design.md D-4). Parsed
# via `while IFS='=' read -r k v`, never `eval` and without a standalone
# `jq` binary -- `gh --jq` is gh's own embedded jq, not a jq we could shell
# out to here. -print-delivery already re-validates both $TOOLCHAIN_ID and
# $TARGET against the same strict formats scripts/toolbump's real run
# enforces (E6-equivalent target format, ^[a-z0-9]+$ id format) before
# printing anything, so a malformed manifest value can only ever be
# something this script then uses as inert, literal commit/PR text --
# never executed, never eval'd.
BRANCH=""
COMMIT=""
PR_TITLE=""
while IFS='=' read -r key value; do
  case "$key" in
    branch) BRANCH="$value" ;;
    commit) COMMIT="$value" ;;
    pr_title) PR_TITLE="$value" ;;
  esac
done < <(go run ./scripts/toolbump -toolchain="$TOOLCHAIN_ID" -target="$TARGET" -print-delivery)

if [ -z "$BRANCH" ] || [ -z "$COMMIT" ] || [ -z "$PR_TITLE" ]; then
  echo "::error::failed to read a complete branch/commit/pr_title manifest from scripts/toolbump -print-delivery" >&2
  exit 1
fi

while IFS= read -r file; do
  [ -n "$file" ] && git add -- "$file"
done <<<"$CHANGED_FILES"

git commit -m "$COMMIT"

# --- Push via an explicit throwaway-branch refspec -- never a direct push
# to develop/main (Threat Matrix: push state). Both branches require every
# change to go through a pull request (branch protection with
# enforce_admins:true, same as release.yml:618-625's CHANGELOG PR), so
# there is no ref this script could push a commit onto directly even if it
# tried. `--force` is safe here specifically because $BRANCH is a
# namespaced, single-purpose branch this script owns and recreates fresh on
# every run -- never a shared branch.
git push --force origin "HEAD:refs/heads/${BRANCH}"

# --- Idempotent PR (Threat Matrix: PR commands). $BRANCH derives only from
# already-validated $TOOLCHAIN_ID/$TARGET, so there is no argument-injection
# surface here.
EXISTING_PR="$(gh pr list --head "$BRANCH" --base develop --state open --json number --jq '.[0].number')"
if [ -n "$EXISTING_PR" ]; then
  echo "ℹ️ PR #${EXISTING_PR} already open for ${BRANCH} -- branch updated, nothing else to do."
else
  gh pr create \
    --base develop \
    --head "$BRANCH" \
    --title "$PR_TITLE" \
    --body "Automated ${TOOLCHAIN_ID} toolchain bump to ${TARGET}, generated by \`make bump-toolchain\` (scripts/toolbump). Validated by \`ci-final\` before this PR was opened."
  echo "✅ Opened ${TOOLCHAIN_ID} toolchain bump PR for ${TARGET}"
fi
