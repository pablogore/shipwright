#!/bin/bash

# changelog-prepend-release_test.sh - Regression tests for
# changelog-prepend-release.sh
#
# Behavioral, not text-grep: each case writes real files to a temp
# directory and asserts on the actual resulting CHANGELOG.md content, the
# same self-contained-harness style as final-sha-gate-check_test.sh.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREPEND="$SCRIPT_DIR/changelog-prepend-release.sh"

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

expect_count() {
  local description="$1" needle="$2" haystack_file="$3" expected="$4"
  local actual
  actual="$(grep -cF "$needle" "$haystack_file")"
  if [ "$actual" -eq "$expected" ]; then
    echo -e "${GREEN}[PASS]${NC} $description (count=$actual)"
  else
    echo -e "${RED}[FAIL]${NC} $description (expected count=$expected, got=$actual)"
    FAILURES=$((FAILURES + 1))
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Case 1: CHANGELOG.md doesn't exist -> created verbatim from the new entry
cat > "$WORKDIR/entry-v1.md" <<'EOF'
## Release v1.0.0

Initial release.
EOF
"$PREPEND" "$WORKDIR/CHANGELOG-missing.md" "$WORKDIR/entry-v1.md" v1.0.0
expect_exit "missing changelog file -> created" 0 "$?"
expect_contains "created file has new entry" "## Release v1.0.0" "$WORKDIR/CHANGELOG-missing.md"

# Case 2: existing file with [Unreleased] -> entry prepended, Unreleased reset
cat > "$WORKDIR/CHANGELOG.md" <<'EOF'
## [Unreleased]

### Added
- something pending

## [0.0.1] - 2024-01-01

- first ever release
EOF
cat > "$WORKDIR/entry-v2.md" <<'EOF'
## Release v2.0.0

- a real change
EOF
"$PREPEND" "$WORKDIR/CHANGELOG.md" "$WORKDIR/entry-v2.md" v2.0.0
expect_exit "existing changelog -> prepend succeeds" 0 "$?"
expect_contains "new entry present" "## Release v2.0.0" "$WORKDIR/CHANGELOG.md"
expect_contains "old release preserved below" "## [0.0.1] - 2024-01-01" "$WORKDIR/CHANGELOG.md"
expect_contains "fresh Unreleased section re-opened" "## [Unreleased]" "$WORKDIR/CHANGELOG.md"
new_entry_line="$(grep -n '^## Release v2.0.0$' "$WORKDIR/CHANGELOG.md" | head -1 | cut -d: -f1)"
old_release_line="$(grep -n '^## \[0.0.1\]' "$WORKDIR/CHANGELOG.md" | head -1 | cut -d: -f1)"
if [ "$new_entry_line" -lt "$old_release_line" ]; then
  echo -e "${GREEN}[PASS]${NC} new entry appears above old release"
else
  echo -e "${RED}[FAIL]${NC} new entry did not end up above old release"
  FAILURES=$((FAILURES + 1))
fi

# Case 3: idempotent -- running again for the SAME tag is a no-op
before_checksum="$(shasum "$WORKDIR/CHANGELOG.md")"
"$PREPEND" "$WORKDIR/CHANGELOG.md" "$WORKDIR/entry-v2.md" v2.0.0
expect_exit "re-run for same tag -> exit 0" 0 "$?"
after_checksum="$(shasum "$WORKDIR/CHANGELOG.md")"
if [ "$before_checksum" = "$after_checksum" ]; then
  echo -e "${GREEN}[PASS]${NC} re-run for same tag did not duplicate the entry"
else
  echo -e "${RED}[FAIL]${NC} re-run for same tag changed CHANGELOG.md (duplication risk)"
  FAILURES=$((FAILURES + 1))
fi
expect_count "exactly one v2.0.0 entry after two runs" "## Release v2.0.0" "$WORKDIR/CHANGELOG.md" 1

# Case 4: a DIFFERENT tag still gets prepended normally
cat > "$WORKDIR/entry-v3.md" <<'EOF'
## Release v3.0.0

- another real change
EOF
"$PREPEND" "$WORKDIR/CHANGELOG.md" "$WORKDIR/entry-v3.md" v3.0.0
expect_contains "different tag's entry is added" "## Release v3.0.0" "$WORKDIR/CHANGELOG.md"
expect_contains "previous tag's entry is retained" "## Release v2.0.0" "$WORKDIR/CHANGELOG.md"

# Case 5: idempotency check must not false-positive on a tag that is a
# textual prefix of an already-recorded tag (e.g. v1.0.1 vs v1.0.10) --
# a substring match here would silently skip a genuinely new release.
PREFIX_DIR="$(mktemp -d)"
cat > "$PREFIX_DIR/CHANGELOG.md" <<'EOF'
## [Unreleased]

### Added
- something pending

## Release v1.0.10

- a change in the newer release

## [0.0.1] - 2024-01-01

- first ever release
EOF
cat > "$PREFIX_DIR/entry-v1.0.1.md" <<'EOF'
## Release v1.0.1

- a change in the older, later-processed release
EOF
"$PREPEND" "$PREFIX_DIR/CHANGELOG.md" "$PREFIX_DIR/entry-v1.0.1.md" v1.0.1
expect_contains "prefix-colliding tag is still added, not falsely treated as duplicate" "## Release v1.0.1" "$PREFIX_DIR/CHANGELOG.md"
expect_contains "the tag it collides with is retained" "## Release v1.0.10" "$PREFIX_DIR/CHANGELOG.md"
rm -rf "$PREFIX_DIR"

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}[SUCCESS]${NC} All changelog-prepend-release.sh scenarios behaved as expected"
  exit 0
else
  echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
  exit 1
fi
