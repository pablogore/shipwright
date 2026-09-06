#!/bin/bash

# final-sha-gate-check_test.sh - Regression tests for final-sha-gate-check.sh
#
# No external test framework is used here (none exists in this repo for
# shell scripts); this is a small self-contained harness asserting the exit
# code of final-sha-gate-check.sh under the scenarios required by the
# Real-Dagger Integration Gate contract, including the "skipped/cancelled
# must not produce a false success" fail-closed requirement.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE_CHECK="$SCRIPT_DIR/final-sha-gate-check.sh"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

FAILURES=0

run_case() {
    local description="$1"
    local validation_result="$2"
    local dagger_result="$3"
    local expected_exit="$4"

    VALIDATION_RESULT="$validation_result" DAGGER_INTEGRATION_RESULT="$dagger_result" "$GATE_CHECK" >/dev/null 2>&1
    local actual_exit=$?

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo -e "${GREEN}[PASS]${NC} $description (validation=$validation_result, dagger=$dagger_result, exit=$actual_exit)"
    else
        echo -e "${RED}[FAIL]${NC} $description (validation=$validation_result, dagger=$dagger_result, expected exit=$expected_exit, got=$actual_exit)"
        FAILURES=$((FAILURES + 1))
    fi
}

run_case "both succeed -> gate success"                 "success"   "success"   0
run_case "portable green, integration red -> gate fails" "success"   "failure"   1
run_case "portable red, integration green -> gate fails" "failure"   "success"   1
run_case "integration skipped -> gate fails (fail closed)" "success" "skipped"   1
run_case "integration cancelled -> gate fails (fail closed)" "success" "cancelled" 1
run_case "both fail -> gate fails"                       "failure"   "failure"   1

echo ""
if [ "$FAILURES" -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} All final-sha-gate-check.sh scenarios behaved as expected"
    exit 0
else
    echo -e "${RED}[ERROR]${NC} $FAILURES scenario(s) did not behave as expected"
    exit 1
fi
