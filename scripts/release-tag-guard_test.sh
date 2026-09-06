#!/bin/bash

# release-tag-guard_test.sh - Regression tests for release-tag-guard.sh
#
# Behavioral, not text-grep: each case spins up a real local bare repo as
# `origin`, clones it, and runs the actual script against real git tag/push
# operations. No external test framework is used here (none exists in this
# repo for shell scripts), mirroring final-sha-gate-check_test.sh's
# self-contained harness.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="$SCRIPT_DIR/release-tag-guard.sh"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

FAILURES=0

new_sandbox() {
  local sandbox
  sandbox="$(mktemp -d)"
  git init --quiet --bare "$sandbox/origin.git"
  git clone --quiet "$sandbox/origin.git" "$sandbox/work"
  (
    cd "$sandbox/work"
    git config user.name "test"
    git config user.email "test@example.com"
    echo "first" > file.txt
    git add file.txt
    git commit --quiet -m "first commit"
    git push --quiet origin HEAD:refs/heads/main
  )
  echo "$sandbox"
}

expect_exit() {
  local description="$1" expected="$2" actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    echo -e "${GREEN}[PASS]${NC} $description (exit=$actual)"
  else
    echo -e "${RED}[FAIL]${NC} $description (expected exit=$expected, got=$actual)"
    FAILURES=$((FAILURES + 1))
  fi
}

expect_eq() {
  local description="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    echo -e "${GREEN}[PASS]${NC} $description"
  else
    echo -e "${RED}[FAIL]${NC} $description (expected '$expected', got '$actual')"
    FAILURES=$((FAILURES + 1))
  fi
}

# Case 1: tag doesn't exist -> created and pushed
sandbox="$(new_sandbox)"
(cd "$sandbox/work" && "$GUARD" v1.0.0) >/dev/null 2>&1
expect_exit "nonexistent tag -> create+push, exit 0" 0 "$?"
pushed_commit="$(git -C "$sandbox/origin.git" rev-parse v1.0.0^{commit} 2>/dev/null || echo "MISSING")"
work_commit="$(cd "$sandbox/work" && git rev-parse HEAD)"
expect_eq "created tag pushed to origin points at HEAD" "$work_commit" "$pushed_commit"
rm -rf "$sandbox"

# Case 2: tag exists at current commit -> safe retry, no-op
sandbox="$(new_sandbox)"
(cd "$sandbox/work" && "$GUARD" v1.0.0) >/dev/null 2>&1
(cd "$sandbox/work" && "$GUARD" v1.0.0) >/dev/null 2>&1
expect_exit "same-commit retry -> exit 0" 0 "$?"
rm -rf "$sandbox"

# Case 3: tag exists at a DIFFERENT commit -> fail closed, tag NOT moved
sandbox="$(new_sandbox)"
(cd "$sandbox/work" && "$GUARD" v1.0.0) >/dev/null 2>&1
original_commit="$(git -C "$sandbox/origin.git" rev-parse v1.0.0^{commit})"
(
  cd "$sandbox/work"
  echo "second" > file2.txt
  git add file2.txt
  git commit --quiet -m "second commit"
  git push --quiet origin HEAD:refs/heads/main
)
(cd "$sandbox/work" && "$GUARD" v1.0.0) >/dev/null 2>&1
guard_exit=$?
expect_exit "tag/commit conflict -> fail closed, exit 1" 1 "$guard_exit"
after_commit="$(git -C "$sandbox/origin.git" rev-parse v1.0.0^{commit})"
expect_eq "conflicting tag was NOT moved in origin" "$original_commit" "$after_commit"
rm -rf "$sandbox"

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}[SUCCESS]${NC} All release-tag-guard.sh scenarios behaved as expected"
  exit 0
else
  echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
  exit 1
fi
