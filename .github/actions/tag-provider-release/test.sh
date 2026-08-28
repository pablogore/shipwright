#!/usr/bin/env bash
# Exercises tag.sh's version-bump decision against real git history in a
# disposable scratch repo -- the only way to prove its commit-message scan
# is scoped to the right range and path, since that logic reads git log
# output rather than any value this script could stub. Run directly:
#   ./.github/actions/tag-provider-release/test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TAG_SH="$SCRIPT_DIR/tag.sh"

FAILURES=0

# run_case sets up a fresh scratch repo + bare remote, applies the given
# setup function, runs tag.sh, and echoes "tagged=<bool> tag=<name>".
run_case() {
  local setup_fn="$1"
  local workdir remote_dir output_file
  workdir=$(mktemp -d)
  remote_dir=$(mktemp -d)

  git init -q --bare "$remote_dir"
  git init -q "$workdir"
  (
    cd "$workdir"
    git config user.email "test@example.com"
    git config user.name "Test"
    git remote add origin "$remote_dir"
    "$setup_fn"
  )

  output_file=$(mktemp)
  (
    cd "$workdir"
    PROVIDER_PATH='providers/rust' \
      TAG_PREFIX='providers/rust/v' \
      SOURCE_BRANCH='develop' \
      GITHUB_OUTPUT="$output_file" \
      bash "$TAG_SH" >/dev/null
  )

  local tagged tag
  tagged=$(grep '^tagged=' "$output_file" | cut -d= -f2)
  tag=$(grep '^tag=' "$output_file" | cut -d= -f2)
  rm -rf "$workdir" "$remote_dir" "$output_file"
  echo "tagged=$tagged tag=$tag"
}

assert_eq() {
  local name="$1" got="$2" want="$3"
  if [ "$got" != "$want" ]; then
    echo "❌ $name: got [$got], want [$want]"
    FAILURES=$((FAILURES + 1))
  else
    echo "✅ $name"
  fi
}

# --- case: provider unchanged -> no tag -------------------------------
setup_unchanged() {
  mkdir -p providers/go
  echo 'go' > providers/go/main.go
  git add -A && git commit -q -m 'feat(providers/go): unrelated change'
}
assert_eq "provider unchanged -> no tag" \
  "$(run_case setup_unchanged)" \
  "tagged=false tag="

# --- case: first tag, no previous tag exists --------------------------
setup_first_tag() {
  mkdir -p providers/rust
  echo 'rust' > providers/rust/main.rs
  git add -A && git commit -q -m 'feat(providers/rust): initial provider'
}
assert_eq "first tag with no previous tag -> 0.1.0" \
  "$(run_case setup_first_tag)" \
  "tagged=true tag=providers/rust/v0.1.0"

# --- case: BREAKING inside the provider path, unrelated commit after --
setup_breaking_in_provider_then_unrelated() {
  mkdir -p providers/rust
  echo 'rust' > providers/rust/main.rs
  git add -A && git commit -q -m 'feat(providers/rust): initial provider'
  git tag providers/rust/v0.1.0

  echo 'rust v2' > providers/rust/main.rs
  git add -A && git commit -q -m 'BREAKING: change the provider public API'

  echo 'readme' > README.md
  git add -A && git commit -q -m 'docs: update README'
}
assert_eq "breaking in provider + later unrelated commit -> major" \
  "$(run_case setup_breaking_in_provider_then_unrelated)" \
  "tagged=true tag=providers/rust/v1.0.0"

# --- case: BREAKING outside the provider path, fix inside it ----------
setup_breaking_outside_fix_inside() {
  mkdir -p providers/rust
  echo 'rust' > providers/rust/main.rs
  git add -A && git commit -q -m 'feat(providers/rust): initial provider'
  git tag providers/rust/v0.1.0

  echo 'core' > core.go
  git add -A && git commit -q -m 'BREAKING: change the core API'

  echo 'rust fix' > providers/rust/main.rs
  git add -A && git commit -q -m 'fix(providers/rust): small bugfix'
}
assert_eq "breaking outside provider + fix inside -> minor, not major" \
  "$(run_case setup_breaking_outside_fix_inside)" \
  "tagged=true tag=providers/rust/v0.2.0"

echo
if [ "$FAILURES" -gt 0 ]; then
  echo "$FAILURES case(s) failed"
  exit 1
fi
echo "All cases passed"
