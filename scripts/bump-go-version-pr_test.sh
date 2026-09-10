#!/bin/bash

# bump-go-version-pr_test.sh - Regression tests for bump-go-version-pr.sh
#
# Behavioral, not text-grep, mirroring changelog-prepend-release_test.sh's
# self-contained-harness style -- but unlike that script, bump-go-version-pr.sh
# talks to `git` (push) and `gh` (PR creation), which this test must never let
# touch the real repo or GitHub. So this harness stubs BOTH `git` and `gh` as
# fake executables placed first on PATH: every invocation is appended to a
# call-log file (so tests can assert exactly which git/gh subcommands and
# flags were used, e.g. "never git add -A"), and canned output for the read
# subcommands (`git status`, `git diff --name-only`, `gh pr list`) is served
# from files the test writes before each run. The script under test never
# sees a real git binary, so it cannot push, commit, or open a PR for real no
# matter what it does internally.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PR_SCRIPT="$SCRIPT_DIR/bump-go-version-pr.sh"

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

expect_contains() {
  local description="$1" needle="$2" haystack_file="$3"
  if grep -qF "$needle" "$haystack_file"; then
    echo -e "${GREEN}[PASS]${NC} $description"
  else
    echo -e "${RED}[FAIL]${NC} $description (did not find: $needle)"
    FAILURES=$((FAILURES + 1))
  fi
}

expect_not_contains() {
  local description="$1" needle="$2" haystack_file="$3"
  if [ -f "$haystack_file" ] && grep -qF "$needle" "$haystack_file"; then
    echo -e "${RED}[FAIL]${NC} $description (unexpectedly found: $needle)"
    FAILURES=$((FAILURES + 1))
  else
    echo -e "${GREEN}[PASS]${NC} $description"
  fi
}

expect_empty_file() {
  local description="$1" file="$2"
  if [ ! -s "$file" ]; then
    echo -e "${GREEN}[PASS]${NC} $description"
  else
    echo -e "${RED}[FAIL]${NC} $description (expected empty, got: $(cat "$file"))"
    FAILURES=$((FAILURES + 1))
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

STUB_BIN="$WORKDIR/bin"
mkdir -p "$STUB_BIN"

# --- Fake `git`: logs every invocation, serves canned output for the read
# subcommands this script relies on, no-ops everything else (add/commit/push
# never touch a real repo).
cat > "$STUB_BIN/git" <<'EOF'
#!/bin/bash
echo "git $*" >> "$GIT_CALL_LOG"
case "$1" in
  status)
    [ -f "$GIT_STATUS_PORCELAIN_FILE" ] && cat "$GIT_STATUS_PORCELAIN_FILE"
    exit 0
    ;;
  diff)
    [ -f "$GIT_DIFF_NAME_ONLY_FILE" ] && cat "$GIT_DIFF_NAME_ONLY_FILE"
    exit 0
    ;;
  add|commit|push)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
EOF
chmod +x "$STUB_BIN/git"

# --- Fake `gh`: logs every invocation, serves canned `gh pr list` output,
# no-ops `gh pr create` (never opens a real PR).
cat > "$STUB_BIN/gh" <<'EOF'
#!/bin/bash
echo "gh $*" >> "$GH_CALL_LOG"
if [ "$1" = "pr" ] && [ "$2" = "list" ]; then
  [ -f "$GH_PR_LIST_OUTPUT_FILE" ] && cat "$GH_PR_LIST_OUTPUT_FILE"
  exit 0
fi
exit 0
EOF
chmod +x "$STUB_BIN/gh"

export PATH="$STUB_BIN:$PATH"

# A valid "repo root" fixture: bump-go-version-pr.sh's repo-root guard (like
# scripts/gobump's requireRepoRoot) requires go.work + .dagger as direct
# children of cwd -- both are plain filesystem checks, never git/gh calls.
new_repo_root() {
  local root
  root="$(mktemp -d)"
  : > "$root/go.work"
  mkdir -p "$root/.dagger"
  echo "$root"
}

# run_case <cwd> <target> <git_status_porcelain_content> <git_diff_name_only_content> <gh_pr_list_output_content>
# Writes the canned fixture files, runs the script from <cwd>, and returns
# its exit code via $? (caller reads $GIT_CALL_LOG / $GH_CALL_LOG afterward).
run_case() {
  local cwd="$1" target="$2" status_content="$3" diff_content="$4" pr_list_content="$5"

  GIT_CALL_LOG="$(mktemp)"
  GH_CALL_LOG="$(mktemp)"
  GIT_STATUS_PORCELAIN_FILE="$(mktemp)"
  GIT_DIFF_NAME_ONLY_FILE="$(mktemp)"
  GH_PR_LIST_OUTPUT_FILE="$(mktemp)"

  printf '%s' "$status_content" > "$GIT_STATUS_PORCELAIN_FILE"
  printf '%s' "$diff_content" > "$GIT_DIFF_NAME_ONLY_FILE"
  printf '%s' "$pr_list_content" > "$GH_PR_LIST_OUTPUT_FILE"

  export GIT_CALL_LOG GH_CALL_LOG GIT_STATUS_PORCELAIN_FILE GIT_DIFF_NAME_ONLY_FILE GH_PR_LIST_OUTPUT_FILE

  (cd "$cwd" && "$PR_SCRIPT" "$target") >"$WORKDIR/last-stdout" 2>"$WORKDIR/last-stderr"
  return $?
}

# ============================================================================
# Case 5.6: non-repo-root cwd -> refuses before any git/gh call
# ============================================================================
NON_REPO="$(mktemp -d)"
run_case "$NON_REPO" "1.26.7" "" "" ""
exit_code=$?
expect_exit "non-repo-root cwd -> refuses" 1 "$exit_code"
expect_empty_file "non-repo-root cwd -> zero git calls before refusing" "$GIT_CALL_LOG"
expect_empty_file "non-repo-root cwd -> zero gh calls before refusing" "$GH_CALL_LOG"
rm -rf "$NON_REPO"

# ============================================================================
# Case 5.2: dirty worktree (untracked files present) -> refuses, no commit
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "1.26.7" "?? some-stray-file.txt" "" ""
exit_code=$?
expect_exit "untracked files in worktree -> refuses" 1 "$exit_code"
expect_not_contains "dirty worktree -> no commit attempted" "git commit" "$GIT_CALL_LOG"
expect_not_contains "dirty worktree -> no push attempted" "git push" "$GIT_CALL_LOG"
expect_empty_file "dirty worktree -> zero gh calls" "$GH_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Case 5.3: commit stages exactly the planned + Upgrade-touched paths --
# never `git add -A` / `git commit -a`
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "1.26.7" " M go.mod
 M go.work
 M .go-version" "go.mod
go.work
.go-version" ""
exit_code=$?
expect_exit "clean-of-untracked worktree -> proceeds" 0 "$exit_code"
expect_contains "stages go.mod individually" "git add -- go.mod" "$GIT_CALL_LOG"
expect_contains "stages go.work individually" "git add -- go.work" "$GIT_CALL_LOG"
expect_contains "stages .go-version individually" "git add -- .go-version" "$GIT_CALL_LOG"
expect_not_contains "never uses git add -A" "git add -A" "$GIT_CALL_LOG"
expect_not_contains "never uses git add ." "git add ." "$GIT_CALL_LOG"
expect_not_contains "never uses git commit -a" "git commit -a" "$GIT_CALL_LOG"
expect_contains "commit message references the target version" "chore(go): bump Go toolchain to 1.26.7" "$GIT_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Case 5.4: push uses the exact refspec HEAD:refs/heads/chore/go-<target>,
# never a direct push to develop/main
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "1.26.7" " M go.mod" "go.mod" ""
exit_code=$?
expect_contains "push uses the exact throwaway-branch refspec" "git push --force origin HEAD:refs/heads/chore/go-1.26.7" "$GIT_CALL_LOG"
expect_not_contains "never pushes refs/heads/develop" "refs/heads/develop" "$GIT_CALL_LOG"
expect_not_contains "never pushes refs/heads/main" "refs/heads/main" "$GIT_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Case 5.5a: gh pr create invoked with exact flags when no PR exists yet
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "1.26.7" " M go.mod" "go.mod" ""
expect_contains "gh pr list queried for the throwaway branch" "gh pr list --head chore/go-1.26.7 --base develop" "$GH_CALL_LOG"
expect_contains "gh pr create invoked with exact --base/--head flags" "gh pr create --base develop --head chore/go-1.26.7" "$GH_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Case 5.5b: idempotent re-run -- an already-open PR for this branch means
# no duplicate is created
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "1.26.7" " M go.mod" "go.mod" "42"
exit_code=$?
expect_exit "re-run with existing open PR -> exit 0" 0 "$exit_code"
expect_contains "still queries gh pr list on re-run" "gh pr list --head chore/go-1.26.7 --base develop" "$GH_CALL_LOG"
expect_not_contains "re-run does not open a duplicate PR" "gh pr create" "$GH_CALL_LOG"
rm -rf "$REPO"

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}[SUCCESS]${NC} All bump-go-version-pr.sh scenarios behaved as expected"
  exit 0
else
  echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
  exit 1
fi
