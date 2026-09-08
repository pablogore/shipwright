## Release v0.12.0

**Release Date:** 2026-09-08

### Changes since v0.11.0


#### 🐛 Bug Fixes
- fix(release): resolve release.yml conflict merging main into develop (#266) (b7d54ff)
- fix(release): open a small PR for CHANGELOG.md instead of pushing to main (#262) (1a62f95)
- fix(release): push CHANGELOG commit as HEAD:main, not main (#260) (285f667)
- Merge pull request #258 from pablogore/fix/ci-dagger-engine-version-0219 (c201424)
- fix(ci): bump pinned DAGGER_VERSION to v0.21.9 (f07752b)


#### 📝 Other Changes
- Merge pull request #269 from pablogore/develop (98659d8)
- Merge pull request #268 from pablogore/fix/real-merge-commit-2 (c7dafa3)
- Merge remote-tracking branch 'origin/main' into fix/real-merge-commit-2 (11048ba)
- chore(release): real merge commit of main into develop (#267) (e941016)
- Merge main into develop (attempt 2, real merge commit) (#265) (a11f4aa)
- Merge main into develop, resolve CHANGELOG-step conflict (#264) (f25875c)


---

# Changelog

All notable changes to Shipwright will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2025-09-14

**⚠️ Legacy release — imported from pre-rebrand history, does not meet the current release-integrity contract (RELEASE-DIST-01A).**

- GitHub Release published 2025-09-14 (02:04 UTC), `target_commitish: develop` — a branch name, not a commit SHA.
- No corresponding Git tag exists in the current repository history; `v0.5.0` cannot be resolved against any commit today.
- Published under the legacy `syntegrity-dagger-*` binary naming convention (pre-rebrand), not the current `shipwright-*` distribution contract.
- Contents cannot be verified against current Git history. No feature list is reconstructed here beyond what is stated above — see [Links](#links) for the raw GitHub Release.

## [0.4.1] - 2025-09-14

**⚠️ Legacy release — imported from pre-rebrand history, does not meet the current release-integrity contract (RELEASE-DIST-01A).**

- GitHub Release published 2025-09-14 (01:52 UTC), `target_commitish: develop` — a branch name, not a commit SHA.
- No corresponding Git tag exists in the current repository history; `v0.4.1` cannot be resolved against any commit today.
- Published under the legacy `syntegrity-dagger-*` binary naming convention (pre-rebrand), not the current `shipwright-*` distribution contract.
- Contents cannot be verified against current Git history. No feature list is reconstructed here beyond what is stated above.

## [0.4.0-beta.1] - 2025-09-14

**⚠️ Legacy pre-release — imported from pre-rebrand history, does not meet the current release-integrity contract (RELEASE-DIST-01A).**

- GitHub Release published 2025-09-14 (01:40 UTC) as a pre-release, `target_commitish: develop` — a branch name, not a commit SHA.
- No corresponding Git tag exists in the current repository history; `v0.4.0-beta.1` cannot be resolved against any commit today.
- Commit `3119fa5` ("fix(workflow): fix artifact handling in release workflows") is reachable from current `develop`/`main` and lands 1m38s before this release's publish timestamp — a timing correlation, not proof. No tag ever bound this release to that or any other commit, so the Tag↔Commit link cannot be confirmed.
- Published under the legacy `syntegrity-dagger-*` binary naming convention (pre-rebrand), not the current `shipwright-*` distribution contract.

## [0.0.2] - 2024-01-15

### Added
- Release pipeline with multi-platform builds
- GitHub Actions workflow for automated releases
- Binary distribution for Linux, macOS, and Windows
- Installation script for easy setup
- Shared library documentation

### Changed
- Simplified build process for pipeline environment
- Updated release workflow permissions
- Improved container registry integration

### Fixed
- Resolved merge conflicts in release workflow
- Fixed build permissions and token configuration
- Corrected release pipeline configuration

## [0.0.1] - 2024-01-10

### Added
- Initial release of Shipwright
- Core pipeline architecture with dependency injection
- Go-Kit pipeline implementation
- Docker-Go pipeline implementation
- Infrastructure pipeline implementation
- Step registry system with pluggable steps
- Hook system for pre/post processing
- Configuration management with YAML support
- Dagger SDK integration for container operations
- CLI interface with comprehensive options
- Test suite with unit and integration tests
- Security scanning integration
- Linting and code quality checks
- Multi-environment support (dev, staging, prod)
- Container registry publishing
- Git repository cloning (SSH/HTTPS)
- Coverage reporting
- Parallel step execution
- Local execution mode
- Comprehensive logging with structured output

### Features
- **Pipeline Types**:
  - Go-Kit pipeline for microservices
  - Docker-Go pipeline for containerized applications
  - Infrastructure pipeline for deployment automation

- **Step Implementations**:
  - Setup step for environment preparation
  - Build step for application compilation
  - Test step for unit and integration testing
  - Lint step for code quality checks
  - Security step for vulnerability scanning
  - Package step for artifact creation
  - Tag step for version management
  - Push step for registry publishing

- **Configuration**:
  - YAML configuration files
  - Environment variable overrides
  - Command-line flag support
  - Multi-layered configuration system

- **CI/CD Integration**:
  - GitHub Actions support
  - GitLab CI support
  - Jenkins pipeline support
  - Custom CI/CD platform support

- **Security**:
  - Vulnerability scanning with govulncheck
  - Dependency auditing
  - Security policy enforcement
  - Container security best practices

- **Performance**:
  - Container caching
  - Parallel execution
  - Incremental builds
  - Resource optimization

### Technical Details
- **Go Version**: 1.25.1
- **Dagger SDK**: v0.18.17
- **Architecture**: Modular with dependency injection
- **Testing**: Unit tests with testify, integration tests with Dagger
- **Linting**: golangci-lint with comprehensive rules
- **Documentation**: Comprehensive API and usage documentation

## [Pre-Release] - 2024-01-01

### Development History
- Initial project setup and architecture design
- Core interfaces and abstractions
- Pipeline registry implementation
- Step registry system
- Hook management system
- Configuration management
- Dagger SDK integration
- CLI interface development
- Test suite implementation
- Documentation creation
- CI/CD pipeline setup
- Security integration
- Performance optimization
- Cross-platform support
- Release automation

---

## Release Notes Format

### Version Numbering
- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (0.X.0): New features (backward compatible)
- **PATCH** (0.0.X): Bug fixes (backward compatible)

### Change Categories
- **Added**: New features
- **Changed**: Changes to existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements

### Breaking Changes
Breaking changes are clearly marked and include:
- Description of the change
- Migration guide
- Timeline for deprecation (if applicable)

### Migration Guides
For major version updates, migration guides are provided to help users upgrade their configurations and code.

---

## Contributing to Changelog

When contributing to the project:

1. Add your changes to the `[Unreleased]` section
2. Use the appropriate category (Added, Changed, Fixed, etc.)
3. Include a brief description of the change
4. Reference related issues or pull requests
5. Follow the existing format and style

### Example Entry

```markdown
### Added
- New pipeline type for Node.js applications (#123)
- Support for custom build arguments in Docker pipeline (#124)

### Changed
- Updated Go version requirement to 1.25.1 (#125)
- Improved error messages for configuration validation (#126)

### Fixed
- Resolved issue with SSH key authentication (#127)
- Fixed memory leak in pipeline execution (#128)
```

---

## Links

- [GitHub Releases](https://github.com/pablogore/shipwright/releases)
- [Documentation](https://github.com/pablogore/shipwright/tree/main/docs)
- [Contributing Guide](CONTRIBUTING.md)
- [API Reference](docs/API.md)

---

## [Unreleased]

### Added
- Changes in progress...
