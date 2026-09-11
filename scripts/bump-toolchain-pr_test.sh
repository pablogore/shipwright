#!/bin/bash

# bump-toolchain-pr_test.sh - Regression tests for bump-toolchain-pr.sh,
# generalized from bump-go-version-pr_test.sh#280 (issue #284): the same 4
# git/gh threat-matrix scenarios (git repository selection, commit state,
# push state, PR commands), now driven by a stubbed `go` binary standing in
# for `go run ./scripts/toolbump ... -print-delivery` instead of a literal
# "go" everywhere.
#
# Behavioral, not text-grep, mirroring changelog-prepend-release_test.sh's
# self-contained-harness style -- but this script talks to `git` (push),
# `gh` (PR creation), and `go` (delivery manifest), which this test must
# never let touch the real repo, GitHub, or build anything. So this harness
# stubs all THREE as fake executables placed first on PATH: every
# invocation is appended to a call-log file (so tests can assert exactly
# which subcommands and flags were used, e.g. "never git add -A"), and
# canned output for the read subcommands (`git status`, `git diff
# --name-only`, `gh pr list`, `go run ./scripts/toolbump ... -print-delivery`)
# is served from files the test writes before each run. The script under
# test never sees a real git/gh/go binary, so it cannot push, commit, open
# a PR, or run any Go code for real no matter what it does internally.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PR_SCRIPT="$SCRIPT_DIR/bump-toolchain-pr.sh"

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

expect_not_exists() {
  local description="$1" path="$2"
  if [ -e "$path" ]; then
    echo -e "${RED}[FAIL]${NC} $description (unexpectedly exists: $path)"
    FAILURES=$((FAILURES + 1))
  else
    echo -e "${GREEN}[PASS]${NC} $description"
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

# --- Fake `go`: logs every invocation, serves the canned -print-delivery
# manifest for `go run ./scripts/toolbump ... -print-delivery` -- the
# script under test never builds or runs real Go code.
cat > "$STUB_BIN/go" <<'EOF'
#!/bin/bash
echo "go $*" >> "$GO_CALL_LOG"
if [ "$1" = "run" ]; then
  [ -f "$GO_TOOLBUMP_MANIFEST_FILE" ] && cat "$GO_TOOLBUMP_MANIFEST_FILE"
  exit 0
fi
exit 0
EOF
chmod +x "$STUB_BIN/go"

export PATH="$STUB_BIN:$PATH"

# A valid "repo root" fixture: bump-toolchain-pr.sh's repo-root guard is a
# single toolchain-agnostic `[ -e .git ]` check (design.md's "Toolchain-
# agnostic root guard" requirement) -- unlike scripts/gobump#280's
# go.work+.dagger pair, so a bare `.git` entry is sufficient here.
new_repo_root() {
  local root
  root="$(mktemp -d)"
  : > "$root/.git"
  echo "$root"
}

default_manifest() {
  printf 'branch=chore/go-1.26.7\ncommit=chore(go): bump Go toolchain to 1.26.7\npr_title=chore(go): bump Go toolchain to 1.26.7\n'
}

# run_case <cwd> <toolchain-id> <target> <git_status_porcelain_content> <git_diff_name_only_content> <gh_pr_list_output_content> <manifest_content>
# Writes the canned fixture files, runs the script from <cwd>, and returns
# its exit code via $? (caller reads $GIT_CALL_LOG / $GH_CALL_LOG / $GO_CALL_LOG afterward).
run_case() {
  local cwd="$1" id="$2" target="$3" status_content="$4" diff_content="$5" pr_list_content="$6" manifest_content="$7"

  GIT_CALL_LOG="$(mktemp)"
  GH_CALL_LOG="$(mktemp)"
  GO_CALL_LOG="$(mktemp)"
  GIT_STATUS_PORCELAIN_FILE="$(mktemp)"
  GIT_DIFF_NAME_ONLY_FILE="$(mktemp)"
  GH_PR_LIST_OUTPUT_FILE="$(mktemp)"
  GO_TOOLBUMP_MANIFEST_FILE="$(mktemp)"

  printf '%s' "$status_content" > "$GIT_STATUS_PORCELAIN_FILE"
  printf '%s' "$diff_content" > "$GIT_DIFF_NAME_ONLY_FILE"
  printf '%s' "$pr_list_content" > "$GH_PR_LIST_OUTPUT_FILE"
  # Always terminated with a trailing newline, regardless of what the
  # `$(...)` command substitution that built manifest_content already
  # stripped: bump-toolchain-pr.sh's `while IFS='=' read -r k v` parser
  # (like any `read`-based line loop) never processes a final line that
  # is not newline-terminated, since `read` returns non-zero at EOF before
  # the loop body runs for that line.
  printf '%s\n' "$manifest_content" > "$GO_TOOLBUMP_MANIFEST_FILE"

  export GIT_CALL_LOG GH_CALL_LOG GO_CALL_LOG GIT_STATUS_PORCELAIN_FILE GIT_DIFF_NAME_ONLY_FILE GH_PR_LIST_OUTPUT_FILE GO_TOOLBUMP_MANIFEST_FILE

  (cd "$cwd" && "$PR_SCRIPT" "$id" "$target") >"$WORKDIR/last-stdout" 2>"$WORKDIR/last-stderr"
  return $?
}

# ============================================================================
# Task 3.5 (threat: git repository selection): missing .git -> rejected
# before any git/gh/go command
# ============================================================================
NON_REPO="$(mktemp -d)"
run_case "$NON_REPO" "go" "1.26.7" "" "" "" "$(default_manifest)"
exit_code=$?
expect_exit "non-repo-root cwd -> refuses" 1 "$exit_code"
expect_empty_file "non-repo-root cwd -> zero git calls before refusing" "$GIT_CALL_LOG"
expect_empty_file "non-repo-root cwd -> zero gh calls before refusing" "$GH_CALL_LOG"
expect_empty_file "non-repo-root cwd -> zero go calls before refusing" "$GO_CALL_LOG"
rm -rf "$NON_REPO"

# ============================================================================
# Task 3.8 (threat: push state), part 1: a shell metacharacter in the
# toolchain-id argument -> refuses before any git/gh/go command, never
# reaches the push refspec
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" 'go; rm -rf /' "1.26.7" "" "" "" "$(default_manifest)"
exit_code=$?
expect_exit "metachar in toolchain id -> refuses" 1 "$exit_code"
expect_empty_file "metachar in toolchain id -> zero git calls before refusing" "$GIT_CALL_LOG"
expect_empty_file "metachar in toolchain id -> zero go calls before refusing" "$GO_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.8 (threat: push state), part 2: a shell metacharacter in the
# target argument -> refuses before any git/gh/go command, never reaches
# the push refspec
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" '1.26.7; rm -rf /' "" "" "" "$(default_manifest)"
exit_code=$?
expect_exit "metachar in target -> refuses" 1 "$exit_code"
expect_empty_file "metachar in target -> zero git calls before refusing" "$GIT_CALL_LOG"
expect_empty_file "metachar in target -> zero go calls before refusing" "$GO_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.7 (threat: commit state), part 1: dirty worktree (untracked files
# present) -> refuses, no commit
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" "?? some-stray-file.txt" "" "" "$(default_manifest)"
exit_code=$?
expect_exit "untracked files in worktree -> refuses" 1 "$exit_code"
expect_not_contains "dirty worktree -> no commit attempted" "git commit" "$GIT_CALL_LOG"
expect_not_contains "dirty worktree -> no push attempted" "git push" "$GIT_CALL_LOG"
expect_empty_file "dirty worktree -> zero gh calls" "$GH_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.7 (threat: commit state), part 2: empty diff -> exit 0, nothing
# staged or committed
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" "" "" "" "$(default_manifest)"
exit_code=$?
expect_exit "empty diff -> exits 0, nothing to commit" 0 "$exit_code"
expect_not_contains "empty diff -> no add attempted" "git add" "$GIT_CALL_LOG"
expect_not_contains "empty diff -> no commit attempted" "git commit" "$GIT_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.7 (threat: commit state), part 3: commit stages exactly the
# planned + Upgrade-touched paths -- never `git add -A` / `git commit -a`
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" " M go.mod
 M go.work
 M .go-version" "go.mod
go.work
.go-version" "" "$(default_manifest)"
exit_code=$?
expect_exit "clean-of-untracked worktree -> proceeds" 0 "$exit_code"
expect_contains "stages go.mod individually" "git add -- go.mod" "$GIT_CALL_LOG"
expect_contains "stages go.work individually" "git add -- go.work" "$GIT_CALL_LOG"
expect_contains "stages .go-version individually" "git add -- .go-version" "$GIT_CALL_LOG"
expect_not_contains "never uses git add -A" "git add -A" "$GIT_CALL_LOG"
expect_not_contains "never uses git add ." "git add ." "$GIT_CALL_LOG"
expect_not_contains "never uses git commit -a" "git commit -a" "$GIT_CALL_LOG"
expect_contains "commit message comes from the toolbump manifest" "chore(go): bump Go toolchain to 1.26.7" "$GIT_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.8 (threat: push state), part 3: push uses the exact refspec
# HEAD:refs/heads/chore/<id>-<target> from the manifest, never a direct
# push to develop/main
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" " M go.mod" "go.mod" "" "$(default_manifest)"
exit_code=$?
expect_contains "push uses the exact throwaway-branch refspec" "git push --force origin HEAD:refs/heads/chore/go-1.26.7" "$GIT_CALL_LOG"
expect_not_contains "never pushes refs/heads/develop" "refs/heads/develop" "$GIT_CALL_LOG"
expect_not_contains "never pushes refs/heads/main" "refs/heads/main" "$GIT_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.6 (threat: PR commands), part 1: gh pr create invoked with exact
# flags when no PR exists yet, --title taken verbatim from the manifest
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" " M go.mod" "go.mod" "" "$(default_manifest)"
expect_contains "gh pr list queried for the manifest branch" "gh pr list --head chore/go-1.26.7 --base develop" "$GH_CALL_LOG"
expect_contains "gh pr create invoked with exact --base/--head/--title flags" "gh pr create --base develop --head chore/go-1.26.7 --title chore(go): bump Go toolchain to 1.26.7" "$GH_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.6 (threat: PR commands), part 2: idempotent re-run -- an
# already-open PR for this branch means no duplicate is created
# ============================================================================
REPO="$(new_repo_root)"
run_case "$REPO" "go" "1.26.7" " M go.mod" "go.mod" "42" "$(default_manifest)"
exit_code=$?
expect_exit "re-run with existing open PR -> exit 0" 0 "$exit_code"
expect_contains "still queries gh pr list on re-run" "gh pr list --head chore/go-1.26.7 --base develop" "$GH_CALL_LOG"
expect_not_contains "re-run does not open a duplicate PR" "gh pr create" "$GH_CALL_LOG"
rm -rf "$REPO"

# ============================================================================
# Task 3.6 (threat: PR commands), part 3: a manifest value carrying shell
# metacharacters -- including a double quote, the character that would
# break out of a naive `eval "git commit -m \"$COMMIT\""`-style quoting --
# stays inert, used only as literal --title/--body text, never executed. If
# the script ever used `eval` on the manifest, this payload would create
# $INJECTION_MARKER; the script must never do that.
# ============================================================================
REPO="$(new_repo_root)"
INJECTION_MARKER="$WORKDIR/pwned-marker"
MALICIOUS_TEXT='chore(go): bump"; touch '"${INJECTION_MARKER}"'; echo "pwned'
MALICIOUS_MANIFEST="branch=chore/go-1.26.7
commit=${MALICIOUS_TEXT}
pr_title=${MALICIOUS_TEXT}"
run_case "$REPO" "go" "1.26.7" " M go.mod" "go.mod" "" "$MALICIOUS_MANIFEST"
exit_code=$?
expect_exit "malicious manifest value -> still delivers normally" 0 "$exit_code"
expect_not_exists "manifest value with shell metacharacters is never executed" "$INJECTION_MARKER"
expect_contains "malicious commit text is used verbatim, not executed" "$MALICIOUS_TEXT" "$GIT_CALL_LOG"
rm -rf "$REPO"

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}[SUCCESS]${NC} All bump-toolchain-pr.sh scenarios behaved as expected"
  exit 0
else
  echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
  exit 1
fi
