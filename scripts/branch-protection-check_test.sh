#!/bin/bash

# branch-protection-check_test.sh - Contract tests for branch-protection-check.sh
#
# These are the exact scenarios the old heuristic-based check could not tell
# apart reliably: a merge commit, a squash merge, and a rebase merge must all
# pass, while a direct push must fail -- and none of that may depend on
# commit message, parent count, push commit count, or author, because the
# script's only inputs are the branch name and the GitHub API's own
# commit-to-PR association. Fixture JSON stands in for that API response so
# these run offline, with no live GitHub call.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK="$SCRIPT_DIR/branch-protection-check.sh"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

FAILURES=0

expect_exit() {
  local description="$1" expected="$2" actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    echo -e "${GREEN}[PASS]${NC} $description (exit=$actual)"
  else
    echo -e "${RED}[FAIL]${NC} $description (expected exit=$expected, got=$actual)"
    FAILURES=$((FAILURES + 1))
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Case 1: valid merge commit -- a PR merged with the default "Create a merge
# commit" strategy. GitHub's pulls-for-commit response shape is identical
# across merge strategies; what makes this THE merge-commit scenario is that
# it's the one associated with a multi-parent commit in real history -- a
# fact this script never looks at, by design.
cat > "$WORKDIR/merge-commit.json" <<'EOF'
[
  {
    "number": 100,
    "merged_at": "2026-01-01T00:00:00Z",
    "base": { "ref": "develop" },
    "head": { "ref": "feature/merge-commit-example" }
  }
]
EOF
"$CHECK" develop "$WORKDIR/merge-commit.json" >/dev/null 2>&1
expect_exit "valid merge-commit PR -> allowed" 0 "$?"

# Case 2: valid squash merge -- single resulting commit, no merge parent,
# base is main this time (a non-default protected branch).
cat > "$WORKDIR/squash-merge.json" <<'EOF'
[
  {
    "number": 101,
    "merged_at": "2026-01-02T00:00:00Z",
    "base": { "ref": "main" },
    "head": { "ref": "hotfix/squash-merge-example" }
  }
]
EOF
"$CHECK" main "$WORKDIR/squash-merge.json" >/dev/null 2>&1
expect_exit "valid squash-merge PR -> allowed" 0 "$?"

# Case 3: valid rebase merge -- this is exactly PR #217's shape: single
# parent, single commit, plain conventional-commit message, human author.
# The old heuristics (message pattern, parent count, push commit count,
# author) could not distinguish this from a direct push -- this is the
# false positive that motivated this rewrite. Also asserts an open PR
# entry alongside the merged one doesn't confuse the "pick the merged one"
# selection.
cat > "$WORKDIR/rebase-merge.json" <<'EOF'
[
  {
    "number": 90,
    "merged_at": null,
    "base": { "ref": "develop" },
    "head": { "ref": "some/unrelated-open-pr" }
  },
  {
    "number": 217,
    "merged_at": "2026-09-06T17:57:03Z",
    "base": { "ref": "develop" },
    "head": { "ref": "feat/release-dist-01a-integrity-baseline" }
  }
]
EOF
"$CHECK" develop "$WORKDIR/rebase-merge.json" >/dev/null 2>&1
expect_exit "valid rebase-merge PR -> allowed (ignores unrelated open PR)" 0 "$?"

# Case 4: invalid -- a genuine direct push has no PR associated with the
# commit at all. This must fail closed.
echo '[]' > "$WORKDIR/direct-push.json"
"$CHECK" develop "$WORKDIR/direct-push.json" >/dev/null 2>&1
expect_exit "direct push (no associated PR) -> rejected" 1 "$?"

# Case 5: a PR is associated with the commit but merged into a DIFFERENT
# branch than the one actually pushed to -- must still fail closed rather
# than trusting "some PR, somewhere, merged this commit".
cat > "$WORKDIR/wrong-base.json" <<'EOF'
[
  {
    "number": 50,
    "merged_at": "2026-01-03T00:00:00Z",
    "base": { "ref": "develop" },
    "head": { "ref": "feature/other" }
  }
]
EOF
"$CHECK" main "$WORKDIR/wrong-base.json" >/dev/null 2>&1
expect_exit "merged PR with mismatched base branch -> rejected" 1 "$?"

# Case 6: a PR is associated with the commit but still open (not merged) --
# an open PR referencing a commit is not evidence of a completed merge.
cat > "$WORKDIR/open-pr.json" <<'EOF'
[
  {
    "number": 51,
    "merged_at": null,
    "base": { "ref": "develop" },
    "head": { "ref": "feature/still-open" }
  }
]
EOF
"$CHECK" develop "$WORKDIR/open-pr.json" >/dev/null 2>&1
expect_exit "open (unmerged) PR -> rejected" 1 "$?"

# Case 7: malformed API response (e.g. a truncated/failed API call) must
# fail closed, not be silently treated as "no evidence, but let's allow it".
echo 'not valid json' > "$WORKDIR/malformed.json"
"$CHECK" develop "$WORKDIR/malformed.json" >/dev/null 2>&1
expect_exit "malformed API response -> rejected" 1 "$?"

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}[SUCCESS]${NC} All branch-protection-check.sh scenarios behaved as expected"
  exit 0
else
  echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
  exit 1
fi
