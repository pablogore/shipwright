# Release Process

This document describes the automated release process for Syntegrity Dagger.

## Overview

The project uses an automated release system that:
- Automatically creates releases when merging to `main` branch
- Generates version numbers based on conventional commits
- Creates pre-releases from `develop` branch
- Builds multi-platform binaries
- Generates changelogs automatically

## Branch Strategy

- **`develop`**: Development branch - creates pre-releases
- **`main`**: Production branch - creates stable releases

## Conventional Commits

The release system uses [Conventional Commits](https://www.conventionalcommits.org/) to determine version bumps:

### Commit Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | Minor (0.1.0) |
| `fix` | Bug fix | Patch (0.0.1) |
| `docs` | Documentation changes | Patch (0.0.1) |
| `style` | Code style changes | Patch (0.0.1) |
| `refactor` | Code refactoring | Patch (0.0.1) |
| `perf` | Performance improvements | Patch (0.0.1) |
| `test` | Test changes | Patch (0.0.1) |
| `chore` | Build/tooling changes | Patch (0.0.1) |

### Breaking Changes

Add `!` after the type/scope to trigger a major version bump:

```
feat!: remove deprecated API
fix(api)!: change response format
```

## Release Types

### 1. Stable Releases (from `main`)

**Trigger**: Push to `main` branch or merge PR to `main`

**Process**:
1. Analyzes commits since last tag
2. Determines version bump based on conventional commits
3. Creates git tag
4. Builds binaries for all platforms
5. Creates GitHub release

**Example**:
```bash
# Current version: v0.0.5
# Commits since last tag:
# - feat: add new authentication system
# - fix: resolve memory leak
# Result: v0.1.0 (minor bump due to feat)
```

### 2. Pre-releases (from `develop`)

**Trigger**: Push to `develop` branch

**Process**:
1. Increments minor version
2. Adds pre-release suffix (beta.1, beta.2, etc.)
3. Creates git tag
4. Builds binaries for all platforms
5. Creates GitHub pre-release

**Example**:
```bash
# Current version: v0.0.5
# Push to develop
# Result: v0.1.0-beta.1
```

### 3. Manual Releases

**Trigger**: GitHub Actions workflow dispatch

**Options**:
- Specify exact version: `v1.2.3`
- Choose bump type: major, minor, patch, auto

## Using the Commit Helper

Use the provided script to create conventional commits:

```bash
# New feature
./scripts/commit-helper.sh -t feat -s api -m "add user authentication"

# Bug fix
./scripts/commit-helper.sh -t fix -m "resolve memory leak in cache"

# Breaking change
./scripts/commit-helper.sh -t feat -s api -m "remove deprecated endpoint" -b
```

## Version Bump Examples

### From v0.0.x to v0.x.0

To jump from patch versions to minor versions:

1. **Add a new feature**:
   ```bash
   git commit -m "feat: add new pipeline type"
   git push origin develop
   # Creates: v0.1.0-beta.1
   ```

2. **Merge to main**:
   ```bash
   git checkout main
   git merge develop
   git push origin main
   # Creates: v0.1.0
   ```

### From v0.x.0 to v1.0.0

To jump to major version:

1. **Make a breaking change**:
   ```bash
   git commit -m "feat!: remove deprecated API"
   git push origin develop
   # Creates: v1.0.0-beta.1
   ```

2. **Merge to main**:
   ```bash
   git checkout main
   git merge develop
   git push origin main
   # Creates: v1.0.0
   ```

## Workflow Files

- **`.github/workflows/release.yml`**: Stable releases from `main`
- **`.github/workflows/prerelease.yml`**: Pre-releases from `develop`
- **`.github/workflows/ci.yml`**: Continuous integration

## Release Artifacts

Each release includes:
- Binaries for Linux (amd64, arm64)
- Binaries for macOS (amd64, arm64)
- Binaries for Windows (amd64)
- SHA256 checksums
- Changelog
- Release notes

## Troubleshooting

### Release not triggered

1. Check if the push was to the correct branch (`main` for stable, `develop` for pre-release)
2. Verify the workflow files are in `.github/workflows/`
3. Check GitHub Actions permissions

### Wrong version generated

1. Verify commit messages follow conventional commits format
2. Check if breaking changes are marked with `!`
3. Review the version determination logic in the workflow

### Manual release needed

Use the workflow dispatch feature:
1. Go to GitHub Actions
2. Select the release workflow
3. Click "Run workflow"
4. Choose version or bump type

## Best Practices

1. **Use conventional commits** for automatic versioning
2. **Test in develop** before merging to main
3. **Use pre-releases** for testing new features
4. **Mark breaking changes** with `!`
5. **Keep commit messages clear** and descriptive
6. **Review generated changelogs** before release

## Examples

### Feature Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/new-pipeline

# 2. Make changes and commit
./scripts/commit-helper.sh -t feat -s pipeline -m "add docker pipeline support"

# 3. Push to develop
git push origin feature/new-pipeline
# Create PR to develop

# 4. After PR is merged, pre-release is created automatically
# v0.1.0-beta.1

# 5. Test the pre-release
# Download and test the beta version

# 6. When ready, merge develop to main
git checkout main
git merge develop
git push origin main
# Creates stable release: v0.1.0
```

### Hotfix Workflow

```bash
# 1. Create hotfix branch from main
git checkout main
git checkout -b hotfix/critical-bug

# 2. Fix the bug
./scripts/commit-helper.sh -t fix -m "resolve critical security issue"

# 3. Push and create PR to main
git push origin hotfix/critical-bug
# Create PR to main

# 4. After merge, patch release is created
# v0.0.6 (patch bump)
```
