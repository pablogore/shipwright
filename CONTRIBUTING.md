# Contributing to Syntegrity Dagger

Thank you for your interest in contributing to Syntegrity Dagger! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Documentation Guidelines](#documentation-guidelines)
- [Release Process](#release-process)

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- **Go 1.25.1 or later** - [Download Go](https://golang.org/dl/)
- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Make** - For build automation
- **Git** - For version control

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/syntegrity-dagger.git
   cd syntegrity-dagger
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/getsyntegrity/syntegrity-dagger.git
   ```

## Development Setup

### 1. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools
```

### 2. Build the Project

```bash
# Build the binary
make build

# Run tests
make test

# Run linting
make lint

# Run all checks
make check
```

### 3. Verify Installation

```bash
# Test the built binary
./syntegrity-dagger --version
./syntegrity-dagger --help
```

## Contributing Process

### 1. Create a Feature Branch

```bash
# Create and switch to a new branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/issue-description
```

### 2. Make Your Changes

- Write clean, well-documented code
- Follow the established code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run linting
make lint

# Run security checks
make security

# Run all checks
make check
```

### 4. Commit Your Changes

```bash
# Stage your changes
git add .

# Commit with a descriptive message
git commit -m "feat: add new pipeline type for Node.js applications

- Implement NodePipeline with npm support
- Add Node.js specific build steps
- Include package.json validation
- Add tests for Node.js pipeline

Closes #123"
```

### 5. Push and Create Pull Request

```bash
# Push your branch
git push origin feature/your-feature-name
```

Then create a pull request on GitHub with:
- Clear description of changes
- Reference to related issues
- Screenshots or examples if applicable

## Code Style Guidelines

### Go Code Style

We follow standard Go conventions and use `gofmt` and `golangci-lint`:

```bash
# Format code
make fmt

# Run linting
make lint
```

#### Key Style Rules

1. **Use `gofmt`** for code formatting
2. **Follow Go naming conventions**:
   - Exported functions start with capital letters
   - Package names are lowercase
   - Constants use UPPER_CASE
3. **Write clear, self-documenting code**
4. **Add comments for exported functions and types**
5. **Keep functions small and focused**
6. **Use meaningful variable names**

#### Example

```go
// Package mypipeline provides a custom pipeline implementation.
package mypipeline

import (
    "context"
    "fmt"
    
    "dagger.io/dagger"
    "github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
)

// Pipeline represents a custom pipeline implementation.
type Pipeline struct {
    client *dagger.Client
    config pipelines.Config
}

// New creates a new pipeline instance.
func New(client *dagger.Client, cfg pipelines.Config) pipelines.Pipeline {
    return &Pipeline{
        client: client,
        config: cfg,
    }
}

// Execute runs the pipeline with the given context.
func (p *Pipeline) Execute(ctx context.Context) error {
    // Implementation here
    return nil
}
```

### Configuration Files

- Use YAML for configuration files
- Follow consistent indentation (2 spaces)
- Use descriptive key names
- Include comments for complex configurations

### Documentation

- Write clear, concise documentation
- Use proper markdown formatting
- Include code examples
- Keep documentation up-to-date with code changes

## Testing Guidelines

### Unit Testing

Write unit tests for all new functionality:

```go
package mypipeline_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/getsyntegrity/syntegrity-dagger/internal/pipelines"
    "github.com/getsyntegrity/syntegrity-dagger/mocks"
)

func TestPipeline_Execute(t *testing.T) {
    // Arrange
    mockConfig := &mocks.MockConfiguration{}
    mockConfig.On("GetString", "pipeline.name").Return("test")
    
    pipeline := mypipeline.New(nil, mockConfig)
    
    // Act
    err := pipeline.Execute(context.Background())
    
    // Assert
    require.NoError(t, err)
    mockConfig.AssertExpectations(t)
}

func TestPipeline_Execute_Error(t *testing.T) {
    // Test error cases
    t.Run("invalid configuration", func(t *testing.T) {
        // Test implementation
    })
    
    t.Run("context cancellation", func(t *testing.T) {
        // Test implementation
    })
}
```

### Integration Testing

For integration tests that require Docker:

```go
func TestPipeline_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Set up test environment
    ctx := context.Background()
    
    // Create Dagger client
    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
    require.NoError(t, err)
    defer client.Close()
    
    // Test pipeline execution
    pipeline := mypipeline.New(client, testConfig)
    err = pipeline.Execute(ctx)
    require.NoError(t, err)
}
```

### Test Coverage

Maintain high test coverage:

```bash
# Run tests with coverage
make test-coverage

# View coverage report
make coverage-html
```

Aim for:
- **Minimum 80% coverage** for new code
- **100% coverage** for critical paths
- **Test all error conditions**

## Documentation Guidelines

### README Updates

When adding new features:

1. Update the main README.md
2. Add usage examples
3. Update the feature list
4. Include configuration examples

### API Documentation

For new APIs:

1. Add Go doc comments
2. Include usage examples
3. Document parameters and return values
4. Update API reference documentation

### Architecture Documentation

For architectural changes:

1. Update architecture diagrams
2. Document design decisions
3. Update component descriptions
4. Include sequence diagrams for complex flows

## Release Process

### Version Numbering

We use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

Before creating a release:

- [ ] All tests pass
- [ ] Documentation is updated
- [ ] CHANGELOG.md is updated
- [ ] Version numbers are updated
- [ ] Release notes are prepared

### Creating a Release

1. Update version in `go.mod` and other files
2. Update `CHANGELOG.md`
3. Create a release tag:
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```
4. Create a GitHub release with release notes

## Types of Contributions

### Bug Reports

When reporting bugs, include:

- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

### Feature Requests

For new features:

- Clear description of the feature
- Use case and motivation
- Proposed implementation (if applicable)
- Breaking changes (if any)

### Code Contributions

We welcome contributions for:

- New pipeline types
- Additional step implementations
- Bug fixes
- Performance improvements
- Documentation improvements
- Test coverage improvements

### Documentation

Help improve documentation:

- Fix typos and grammar
- Add missing examples
- Improve clarity
- Translate to other languages

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Pull Requests**: Code contributions and reviews

### Asking Questions

When asking questions:

1. Search existing issues and discussions first
2. Provide context and environment details
3. Include relevant code snippets
4. Be specific about what you're trying to achieve

## Recognition

Contributors will be recognized in:

- CONTRIBUTORS.md file
- Release notes
- Project documentation

## License

By contributing to Syntegrity Dagger, you agree that your contributions will be licensed under the same license as the project (MIT License).

## Thank You

Thank you for contributing to Syntegrity Dagger! Your contributions help make this project better for everyone.
