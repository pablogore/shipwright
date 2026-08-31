#!/bin/bash

# action-contract_test.sh - Regression test asserting the public GitHub
# composite Action (.github/actions/shipwright/action.yml) only ever emits
# CLI flags that main.go's parseFlags actually registers.
#
# This exists because --pipeline, --list-pipelines, --only-build,
# --only-test and --skip-push were deleted from the CLI (tasks.md 11.3/11.4,
# --workflow is now the sole entrypoint) while the distributed composite
# Action kept building command lines against them, breaking the public
# GitHub Actions integration by contract. No external test framework exists
# for shell/YAML in this repo, so this is a small self-contained rg-based
# harness, mirroring final-sha-gate-check_test.sh's style.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACTION_FILE="$REPO_ROOT/.github/actions/shipwright/action.yml"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

FAILURES=0

assert_absent() {
    local description="$1"
    local pattern="$2"

    if rg -q -- "$pattern" "$ACTION_FILE"; then
        echo -e "${RED}[FAIL]${NC} $description (found forbidden pattern: $pattern)"
        FAILURES=$((FAILURES + 1))
    else
        echo -e "${GREEN}[PASS]${NC} $description"
    fi
}

assert_present() {
    local description="$1"
    local pattern="$2"

    if rg -q -- "$pattern" "$ACTION_FILE"; then
        echo -e "${GREEN}[PASS]${NC} $description"
    else
        echo -e "${RED}[FAIL]${NC} $description (missing expected pattern: $pattern)"
        FAILURES=$((FAILURES + 1))
    fi
}

# Flags deleted from the CLI (main.go parseFlags no longer registers them) --
# the action must never emit or declare these.
assert_absent "does not emit --pipeline"        "\-\-pipeline"
assert_absent "does not emit --list-pipelines"  "\-\-list-pipelines"
assert_absent "does not emit --only-build"      "\-\-only-build"
assert_absent "does not emit --only-test"       "\-\-only-test"
assert_absent "does not emit --skip-push"       "\-\-skip-push"
assert_absent "does not declare a 'stage' input" "^  stage:"

# The manifest-driven contract the CLI actually supports today.
assert_present "declares a 'workflow' input"    "^  workflow:"
assert_present "declares a 'step' input"        "^  step:"
assert_present "declares a 'list-steps' input"  "^  list-steps:"
assert_present "declares a 'branch' input"      "^  branch:"
assert_present "emits --workflow="              "\-\-workflow="
assert_present "emits --branch="                "\-\-branch="

echo ""
if [ "$FAILURES" -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} action.yml matches the CLI's current flag contract"
    exit 0
else
    echo -e "${RED}[ERROR]${NC} $FAILURES contract assertion(s) failed"
    exit 1
fi
