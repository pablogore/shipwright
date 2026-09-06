#!/bin/bash

# final-sha-gate-check.sh - Compose the Final-SHA production-readiness verdict
#
# A SHA is production-ready only when BOTH the portable validation contract
# (final-sha-validation / `make ci-final`, no Dagger engine required) AND the
# real-Dagger-engine integration contract (final-sha-dagger-integration /
# `make test-integration`) succeed for that exact SHA. Fail-closed by
# construction: any result other than exactly 'success' for either input --
# failure, cancelled, skipped, or missing -- is treated as INVALID.
#
# Usage: VALIDATION_RESULT=<result> DAGGER_INTEGRATION_RESULT=<result> ./scripts/final-sha-gate-check.sh
# <result> is a GitHub Actions job-result string: success|failure|cancelled|skipped

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

if [ -z "${VALIDATION_RESULT:-}" ]; then
    error "VALIDATION_RESULT is required"
    exit 1
fi

if [ -z "${DAGGER_INTEGRATION_RESULT:-}" ]; then
    error "DAGGER_INTEGRATION_RESULT is required"
    exit 1
fi

echo "final-sha-validation:          $VALIDATION_RESULT"
echo "final-sha-dagger-integration:  $DAGGER_INTEGRATION_RESULT"

FAILED=0

if [ "$VALIDATION_RESULT" != "success" ]; then
    error "final-sha-validation did not succeed (result: $VALIDATION_RESULT)"
    FAILED=1
fi

if [ "$DAGGER_INTEGRATION_RESULT" != "success" ]; then
    error "final-sha-dagger-integration did not succeed (result: $DAGGER_INTEGRATION_RESULT)"
    FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
    echo ""
    error "❌ FINAL SHA INVALID"
    error "Both final-sha-validation and final-sha-dagger-integration must succeed for the same SHA."
    error "This SHA must NOT be considered production-ready."
    exit 1
fi

echo ""
success "✅ FINAL SHA VALID"
success "Portable validation (ci-final) and real-Dagger-engine integration (test-integration) both passed for this exact SHA."
