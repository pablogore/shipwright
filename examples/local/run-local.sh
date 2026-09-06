#!/bin/bash
# Quick Local Pipeline Runner
# Copy this script to your project root and run: ./run-local.sh
#
# Usage:
#   ./run-local.sh [pipeline] [step] [coverage] [config-file]
#
# Examples:
#   ./run-local.sh                    # Run full pipeline with defaults
#   ./run-local.sh go-service         # Run go-service pipeline
#   ./run-local.sh go-service build   # Run only build step
#   ./run-local.sh go-service "" 95   # Run with 95% coverage threshold
#   ./run-local.sh go-service "" 90 .my-config.yml  # Use custom config

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PIPELINE="${1:-go-service}"
STEP="${2:-}"  # Optional: specific step (setup, build, test, lint, security)
COVERAGE="${3:-90}"
CONFIG_FILE="${4:-.shipwright.yml}"

echo -e "${BLUE}🏠 Shipwright - Local Pipeline Runner${NC}"
echo "=========================================="
echo ""

# Check if shipwright is installed
if ! command -v shipwright &> /dev/null; then
    echo -e "${RED}❌ shipwright not found${NC}"
    echo ""
    echo "Install it with:"
    echo "  curl -L https://github.com/pablogore/shipwright/releases/latest/download/shipwright-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/') -o shipwright"
    echo "  chmod +x shipwright"
    echo "  sudo mv shipwright /usr/local/bin/"
    echo ""
    exit 1
fi

# Check if Go is installed (required for local execution)
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo ""
    echo "Local execution requires Go to be installed."
    echo "Install Go from: https://golang.org/dl/"
    echo ""
    exit 1
fi

# Show version
echo -e "${BLUE}📋 Version:${NC}"
shipwright --version
echo -e "${CYAN}Go version: $(go version)${NC}"
echo ""

# Check if we're in a Go project
if [ ! -f "go.mod" ]; then
    echo -e "${YELLOW}⚠️  Warning: go.mod not found in current directory${NC}"
    echo "This script should be run from the root of a Go project."
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${YELLOW}⚠️  Config file '$CONFIG_FILE' not found${NC}"
    echo "Creating a basic config file..."
    cat > "$CONFIG_FILE" <<EOF
pipeline:
  name: $PIPELINE
  steps:
    - setup
    - build
    - test
    - lint
    - security
  coverage: $COVERAGE
  verbose: true

environment: dev

git:
  ref: main
  protocol: https

security:
  enable_vuln_check: true
  enable_linting: true
EOF
    echo -e "${GREEN}✅ Created $CONFIG_FILE${NC}"
    echo ""
fi

# Build command
CMD="shipwright --pipeline=$PIPELINE --config=$CONFIG_FILE --coverage=$COVERAGE --local --verbose"

if [ -n "$STEP" ]; then
    echo -e "${BLUE}🎯 Running step: ${CYAN}$STEP${NC}"
    echo ""
    $CMD --step="$STEP"
else
    echo -e "${BLUE}🚀 Running full pipeline: ${CYAN}$PIPELINE${NC}"
    echo ""
    $CMD
fi

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Pipeline completed successfully!${NC}"
    echo ""
    echo -e "${CYAN}💡 Tip: Check coverage/ directory for coverage reports${NC}"
else
    echo ""
    echo -e "${RED}❌ Pipeline failed with exit code: $EXIT_CODE${NC}"
    echo ""
    echo -e "${YELLOW}💡 Troubleshooting:${NC}"
    echo "  - Check that all dependencies are installed: go mod download"
    echo "  - Verify Go version matches project requirements"
    echo "  - Review error messages above for specific issues"
    exit $EXIT_CODE
fi
