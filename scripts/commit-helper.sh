#!/bin/bash

# Conventional Commits Helper Script
# This script helps create conventional commits for automatic versioning

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to show usage
show_usage() {
    print_color $BLUE "Conventional Commits Helper"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -t, --type TYPE       Commit type (feat, fix, docs, style, refactor, perf, test, chore)"
    echo "  -s, --scope SCOPE     Optional scope (e.g., api, ui, config)"
    echo "  -m, --message MSG     Commit message"
    echo "  -b, --breaking        Mark as breaking change"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -t feat -s api -m \"add user authentication\""
    echo "  $0 -t fix -m \"resolve memory leak in cache\""
    echo "  $0 -t feat -s api -m \"add new endpoint\" -b"
    echo ""
    echo "Commit Types:"
    echo "  feat:     A new feature (triggers minor version bump)"
    echo "  fix:      A bug fix (triggers patch version bump)"
    echo "  docs:     Documentation only changes"
    echo "  style:    Changes that do not affect the meaning of the code"
    echo "  refactor: A code change that neither fixes a bug nor adds a feature"
    echo "  perf:     A code change that improves performance"
    echo "  test:     Adding missing tests or correcting existing tests"
    echo "  chore:    Changes to the build process or auxiliary tools"
    echo ""
    echo "Breaking Changes:"
    echo "  Add '!' after the type/scope to trigger major version bump"
    echo "  Example: feat!: remove deprecated API"
}

# Default values
COMMIT_TYPE=""
SCOPE=""
MESSAGE=""
BREAKING=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            COMMIT_TYPE="$2"
            shift 2
            ;;
        -s|--scope)
            SCOPE="$2"
            shift 2
            ;;
        -m|--message)
            MESSAGE="$2"
            shift 2
            ;;
        -b|--breaking)
            BREAKING=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_color $RED "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$COMMIT_TYPE" ]]; then
    print_color $RED "Error: Commit type is required"
    show_usage
    exit 1
fi

if [[ -z "$MESSAGE" ]]; then
    print_color $RED "Error: Commit message is required"
    show_usage
    exit 1
fi

# Validate commit type
VALID_TYPES=("feat" "fix" "docs" "style" "refactor" "perf" "test" "chore")
if [[ ! " ${VALID_TYPES[@]} " =~ " ${COMMIT_TYPE} " ]]; then
    print_color $RED "Error: Invalid commit type '$COMMIT_TYPE'"
    print_color $YELLOW "Valid types: ${VALID_TYPES[*]}"
    exit 1
fi

# Build the commit message
COMMIT_MSG="$COMMIT_TYPE"

# Add scope if provided
if [[ -n "$SCOPE" ]]; then
    COMMIT_MSG="$COMMIT_MSG($SCOPE)"
fi

# Add breaking change indicator
if [[ "$BREAKING" == true ]]; then
    COMMIT_MSG="$COMMIT_MSG!"
fi

# Add colon and message
COMMIT_MSG="$COMMIT_MSG: $MESSAGE"

# Show the commit message
print_color $GREEN "Generated commit message:"
print_color $BLUE "$COMMIT_MSG"
echo ""

# Ask for confirmation
read -p "Do you want to commit with this message? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_color $RED "Error: Not in a git repository"
        exit 1
    fi
    
    # Check if there are staged changes
    if ! git diff --cached --quiet; then
        git commit -m "$COMMIT_MSG"
        print_color $GREEN "✅ Commit created successfully!"
    else
        print_color $YELLOW "No staged changes found. Staging all changes..."
        git add .
        git commit -m "$COMMIT_MSG"
        print_color $GREEN "✅ Commit created successfully!"
    fi
    
    # Show version impact
    echo ""
    print_color $BLUE "Version Impact:"
    case $COMMIT_TYPE in
        "feat")
            if [[ "$BREAKING" == true ]]; then
                print_color $RED "🚨 MAJOR version bump (breaking change)"
            else
                print_color $GREEN "✨ MINOR version bump (new feature)"
            fi
            ;;
        "fix")
            if [[ "$BREAKING" == true ]]; then
                print_color $RED "🚨 MAJOR version bump (breaking change)"
            else
                print_color $YELLOW "🔧 PATCH version bump (bug fix)"
            fi
            ;;
        *)
            if [[ "$BREAKING" == true ]]; then
                print_color $RED "🚨 MAJOR version bump (breaking change)"
            else
                print_color $YELLOW "🔧 PATCH version bump (other changes)"
            fi
            ;;
    esac
else
    print_color $YELLOW "Commit cancelled"
fi
