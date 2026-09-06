#!/bin/bash

# Script to apply branch protection rules using GitHub CLI
# This script requires GitHub CLI (gh) to be installed and authenticated

set -e

REPO="pablogore/shipwright"
RULESET_FILE=".github/rulesets/branch-protection-rules.json"

echo "🛡️  Applying branch protection rules to $REPO"
echo ""

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ Error: GitHub CLI (gh) is not installed"
    echo "   Install it from: https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Error: Not authenticated with GitHub CLI"
    echo "   Run: gh auth login"
    exit 1
fi

# Check if ruleset file exists
if [ ! -f "$RULESET_FILE" ]; then
    echo "❌ Error: Ruleset file not found: $RULESET_FILE"
    exit 1
fi

echo "📋 Ruleset configuration:"
echo "   - Target branches: main, develop"
echo "   - Required approvals: 1"
echo "   - Required status checks: build, test, security"
echo "   - Allowed merge methods: squash"
echo ""

# Apply ruleset to repository
echo "🚀 Applying ruleset..."
gh api repos/$REPO/rulesets \
  --method POST \
  --input "$RULESET_FILE"

if [ $? -eq 0 ]; then
    echo "✅ Branch protection rules applied successfully!"
    echo ""
    echo "📝 Summary:"
    echo "   - PRs to 'main' require: 1 approval + CI success (build, test, security)"
    echo "   - PRs to 'develop' require: 1 approval + CI success (build, test, security)"
    echo "   - Hotfix PRs to 'main' require: 1 approval + CI success"
    echo "   - Feature PRs to 'develop' require: 1 approval + CI success"
    echo ""
    echo "🔍 Verify the rules at:"
    echo "   https://github.com/$REPO/settings/rules"
else
    echo "❌ Error: Failed to apply branch protection rules"
    exit 1
fi

