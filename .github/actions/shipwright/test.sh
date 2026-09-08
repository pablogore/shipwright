#!/usr/bin/env bash
# Exercises the shipwright action's lib/*.sh scripts directly -- the same
# convention as .github/actions/tag-provider-release/test.sh. These scripts
# hold every piece of logic this action's security hardening depends on
# (RELEASE-DIST-01C): pinned-version enforcement, fail-closed platform
# detection, mandatory checksum verification, robust CLI version
# verification, and eval-free argument building. None of it needs a real
# GitHub Actions runtime or a real Shipwright binary to test. Run directly:
#   ./.github/actions/shipwright/test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"

FAILURES=0

assert_eq() {
  local name="$1" got="$2" want="$3"
  if [ "$got" != "$want" ]; then
    echo "❌ $name: got [$got], want [$want]"
    FAILURES=$((FAILURES + 1))
  else
    echo "✅ $name"
  fi
}

assert_fails() {
  local name="$1"
  shift
  if "$@" >/tmp/shipwright-test-out.$$ 2>&1; then
    echo "❌ $name: expected failure, but command succeeded"
    cat /tmp/shipwright-test-out.$$
    FAILURES=$((FAILURES + 1))
  else
    echo "✅ $name"
  fi
  rm -f /tmp/shipwright-test-out.$$
}

assert_succeeds() {
  local name="$1"
  shift
  if "$@" >/tmp/shipwright-test-out.$$ 2>&1; then
    echo "✅ $name"
  else
    echo "❌ $name: expected success, but command failed"
    cat /tmp/shipwright-test-out.$$
    FAILURES=$((FAILURES + 1))
  fi
  rm -f /tmp/shipwright-test-out.$$
}

# --- validate-version.sh ------------------------------------------------

assert_fails "empty version is rejected" \
  "$LIB_DIR/validate-version.sh" ""

assert_fails "version=latest is rejected" \
  bash -c "GITHUB_OUTPUT=/dev/null '$LIB_DIR/validate-version.sh' latest"

out=$(mktemp)
GITHUB_OUTPUT="$out" "$LIB_DIR/validate-version.sh" "v1.2.3"
assert_eq "valid pinned version is accepted" "$(cat "$out")" "version=v1.2.3"
rm -f "$out"

# --- resolve-platform.sh --------------------------------------------------

resolve() {
  local out
  out=$(mktemp)
  GITHUB_OUTPUT="$out" "$LIB_DIR/resolve-platform.sh" "$1" "$2" 2>/dev/null
  cat "$out" | paste -sd' ' -
  rm -f "$out"
}

assert_eq "linux/x86_64 -> linux/amd64" \
  "$(resolve Linux x86_64)" "os=linux arch=amd64 binary=shipwright-linux-amd64"
assert_eq "linux/aarch64 -> linux/arm64" \
  "$(resolve Linux aarch64)" "os=linux arch=arm64 binary=shipwright-linux-arm64"
assert_eq "darwin/x86_64 -> darwin/amd64" \
  "$(resolve Darwin x86_64)" "os=darwin arch=amd64 binary=shipwright-darwin-amd64"
assert_eq "darwin/arm64 -> darwin/arm64" \
  "$(resolve Darwin arm64)" "os=darwin arch=arm64 binary=shipwright-darwin-arm64"
assert_eq "windows/amd64 -> windows/amd64.exe" \
  "$(resolve MINGW64_NT-10.0 x86_64)" "os=windows arch=amd64 binary=shipwright-windows-amd64.exe"
assert_eq "windows/arm64 -> windows/arm64.exe" \
  "$(resolve MINGW64_NT-10.0 aarch64)" "os=windows arch=arm64 binary=shipwright-windows-arm64.exe"

assert_fails "unsupported OS fails closed" \
  bash -c "GITHUB_OUTPUT=/dev/null '$LIB_DIR/resolve-platform.sh' SunOS x86_64"
assert_fails "unsupported arch fails closed" \
  bash -c "GITHUB_OUTPUT=/dev/null '$LIB_DIR/resolve-platform.sh' Linux riscv64"

# --- verify-checksum.sh ---------------------------------------------------

CKDIR=$(mktemp -d)
trap 'rm -rf "$CKDIR"' EXIT
echo -n "hello world" > "$CKDIR/shipwright-linux-amd64"
HASH=$(sha256sum "$CKDIR/shipwright-linux-amd64" 2>/dev/null | awk '{print $1}' || shasum -a 256 "$CKDIR/shipwright-linux-amd64" | awk '{print $1}')
printf '%s  shipwright-linux-amd64\n' "$HASH" > "$CKDIR/checksums.txt"

assert_succeeds "correct checksum verifies" \
  "$LIB_DIR/verify-checksum.sh" "$CKDIR/checksums.txt" "$CKDIR/shipwright-linux-amd64" "shipwright-linux-amd64"

echo -n "tampered" > "$CKDIR/tampered-binary"
printf '%s  shipwright-linux-amd64\n' "$HASH" > "$CKDIR/checksums-mismatch.txt"
assert_fails "tampered binary fails checksum" \
  "$LIB_DIR/verify-checksum.sh" "$CKDIR/checksums-mismatch.txt" "$CKDIR/tampered-binary" "shipwright-linux-amd64"

printf '%s  some-other-file\n' "$HASH" > "$CKDIR/checksums-no-entry.txt"
assert_fails "missing checksum entry fails closed" \
  "$LIB_DIR/verify-checksum.sh" "$CKDIR/checksums-no-entry.txt" "$CKDIR/shipwright-linux-amd64" "shipwright-linux-amd64"

assert_fails "missing checksums.txt fails closed" \
  "$LIB_DIR/verify-checksum.sh" "$CKDIR/does-not-exist.txt" "$CKDIR/shipwright-linux-amd64" "shipwright-linux-amd64"

# --- verify-cli-version.sh -------------------------------------------------

FAKE_BIN=$(mktemp)
chmod +x "$FAKE_BIN"
cat > "$FAKE_BIN" <<'EOF'
#!/usr/bin/env bash
echo "level=INFO msg=\"Shipwright version\" version=1.2.3 go_version=go1.23 os=linux arch=amd64 build_time=2026-01-01T00:00:00Z git_commit=abc123"
EOF

assert_succeeds "matching version verifies" \
  "$LIB_DIR/verify-cli-version.sh" "$FAKE_BIN" "v1.2.3"
assert_fails "mismatched version fails closed" \
  "$LIB_DIR/verify-cli-version.sh" "$FAKE_BIN" "v9.9.9"
rm -f "$FAKE_BIN"

# --- build-args.sh ----------------------------------------------------

build() {
  env -i PATH="$PATH" \
    WORKFLOW="$1" CONFIG="$2" ENV="dev" COVERAGE="90" BRANCH="develop" \
    LIST_STEPS="${3:-false}" STEP="${4:-}" EXECUTOR="" VERBOSE="false" \
    GIT_REF="develop" GIT_AUTH="https" \
    "$LIB_DIR/build-args.sh"
}

DEFAULT_ARGS=$(build ".shipwright/workflow.yaml" ".shipwright.yml")
assert_eq "default args, one per line" "$DEFAULT_ARGS" "$(printf '%s\n' \
  '--workflow=.shipwright/workflow.yaml' \
  '--config=.shipwright.yml' \
  '--env=dev' \
  '--coverage=90' \
  '--branch=develop' \
  '--git-ref=develop' \
  '--git-auth=https')"

DANGEROUS='weird value; rm -rf /tmp/should-not-run && echo pwned $(whoami)'
DANGEROUS_ARGS=$(build ".shipwright/workflow.yaml" "$DANGEROUS")
CONFIG_LINE=$(echo "$DANGEROUS_ARGS" | grep '^--config=')
assert_eq "dangerous config value survives as one unsplit argv element" \
  "$CONFIG_LINE" "--config=${DANGEROUS}"

LIST_STEPS_ARGS=$(build ".shipwright/workflow.yaml" ".shipwright.yml" "true")
assert_eq "list-steps=true emits --list-steps, not --step" \
  "$(echo "$LIST_STEPS_ARGS" | grep -cE '^--list-steps$|^--step=')" "1"

echo
if [ "$FAILURES" -gt 0 ]; then
  echo "$FAILURES case(s) failed"
  exit 1
fi
echo "All cases passed"
