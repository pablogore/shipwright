#!/bin/bash

# branch-protection-check.sh - Decide whether a commit landing on a
# protected branch is backed by a real merged pull request.
#
# This is the authoritative half of the check: given the JSON GitHub itself
# returns from `GET /repos/{owner}/{repo}/commits/{sha}/pulls` (the real
# source of truth for "which PR introduced this commit" -- correct for
# merge-commit, squash, and rebase merges alike), decide whether any entry
# is a merged pull request whose base is exactly this branch.
#
# Why this replaced commit-message/parent-count/author heuristics: a
# --rebase merge keeps the original author, the original commit message
# (no PR number), and a single-parent history -- so it is byte-for-byte
# indistinguishable from a real direct push under those heuristics. That
# false-flagged a legitimate rebase merge (PR #217, commit 0f2a002) as a
# direct push. The GitHub API's commit<->PR association is not fooled by
# merge strategy, so it is the only thing this check trusts.
#
# Whether the `gh api` call that produced the input file even succeeded is
# the caller's responsibility: an API failure is not evidence of a
# legitimate merge and must fail closed there, before this script runs.
#
# Usage: ./scripts/branch-protection-check.sh <branch> <pulls-json-file>
# <pulls-json-file> must contain the raw JSON array from the GitHub
# "list pull requests associated with a commit" endpoint.
# Exit 0: a merged PR with base == <branch> was found.
# Exit 1: no such PR was found (empty array, only open PRs, or PRs merged
#         into a different base all count as "not found").

set -uo pipefail

BRANCH="${1:?branch required}"
PULLS_FILE="${2:?pulls JSON file required}"

if ! MATCH="$(jq -r --arg branch "$BRANCH" \
  '[.[] | select(.merged_at != null and .base.ref == $branch)] | first // empty' \
  "$PULLS_FILE" 2>/dev/null)"; then
  echo "Could not parse the GitHub API response for this commit's associated pull requests -- failing closed." >&2
  exit 1
fi

if [ -z "$MATCH" ]; then
  echo "No merged pull request with base '${BRANCH}' is associated with this commit." >&2
  exit 1
fi

PR_NUMBER="$(echo "$MATCH" | jq -r '.number')"
PR_HEAD="$(echo "$MATCH" | jq -r '.head.ref')"
echo "Merged via PR #${PR_NUMBER} (${PR_HEAD} -> ${BRANCH})"
