#!/bin/bash
# Local CI Script for Syntegrity Dagger
# This script demonstrates how to run a complete CI pipeline locally

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PIPELINE_TYPE="${1:-go-kit}"
ENVIRONMENT="${2:-dev}"
COVERAGE_THRESHOLD="${3:-90}"
CONFIG_FILE="${4:-.syntegrity-dagger.yml}"

echo -e "${BLUE}🏠 Local CI Pipeline${NC}"
echo "===================="
echo "Pipeline: $PIPELINE_TYPE"
echo "Environment: $ENVIRONMENT"
echo "Coverage Threshold: $COVERAGE_THRESHOLD%"
echo "Config: $CONFIG_FILE"
echo ""

# Check if syntegrity-dagger is installed
if ! command -v syntegrity-dagger &> /dev/null; then
    echo -e "${RED}❌ syntegrity-dagger not found${NC}"
    echo "Please install it first:"
    echo "  curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o syntegrity-dagger"
    echo "  chmod +x syntegrity-dagger"
    echo "  sudo mv syntegrity-dagger /usr/local/bin/"
    exit 1
fi

# Show version
echo -e "${BLUE}📋 Syntegrity Dagger Version:${NC}"
syntegrity-dagger --version
echo ""

# Function to run a stage
run_stage() {
    local stage=$1
    echo -e "${YELLOW}▶️  Running stage: $stage${NC}"
    
    if syntegrity-dagger \
        --pipeline="$PIPELINE_TYPE" \
        --step="$stage" \
        --config="$CONFIG_FILE" \
        --env="$ENVIRONMENT" \
        --coverage="$COVERAGE_THRESHOLD" \
        --local \
        --verbose; then
        echo -e "${GREEN}✅ Stage $stage completed${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}❌ Stage $stage failed${NC}"
        return 1
    fi
}

# Run pipeline stages
echo -e "${BLUE}🚀 Starting pipeline execution${NC}"
echo ""

# Stage 1: Setup
if ! run_stage "setup"; then
    echo -e "${RED}❌ Setup failed. Aborting.${NC}"
    exit 1
fi

# Stage 2: Build
if ! run_stage "build"; then
    echo -e "${RED}❌ Build failed. Aborting.${NC}"
    exit 1
fi

# Stage 3: Test
if ! run_stage "test"; then
    echo -e "${RED}❌ Test failed. Aborting.${NC}"
    exit 1
fi

# Stage 4: Lint (optional)
if [ -n "$RUN_LINT" ]; then
    if ! run_stage "lint"; then
        echo -e "${YELLOW}⚠️  Lint failed but continuing${NC}"
    fi
fi

# Stage 5: Security (optional)
if [ -n "$RUN_SECURITY" ]; then
    if ! run_stage "security"; then
        echo -e "${YELLOW}⚠️  Security scan found issues but continuing${NC}"
    fi
fi

# Summary
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✅ Pipeline Completed Successfully   ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Artifacts:"
echo "  - Build artifacts: ./bin/"
echo "  - Test coverage: ./coverage/"
echo ""

# Optional: Open coverage report
if command -v open &> /dev/null && [ -f "coverage/coverage.html" ]; then
    read -p "Open coverage report in browser? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        open coverage/coverage.html
    fi
fi

