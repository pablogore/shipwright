# 🏠 Running Pipelines Locally

This guide shows you how to run and test Shipwright pipelines locally in your project **without needing cloud services or Docker**.

## 🎯 Why Run Locally?

- **Fast feedback**: Test your pipelines before pushing to CI/CD
- **No cloud required**: Run everything on your machine
- **Cost-effective**: No cloud compute costs
- **Debug easily**: Full access to logs and intermediate results
- **Test configurations**: Validate your `.shipwright.yml` before committing

## 🚀 Quick Start

### Option 1: Using the Quick Script (Recommended)

1. **Copy the script to your project:**
   ```bash
   cp examples/local/run-local.sh .
   chmod +x run-local.sh
   ```

2. **Run the pipeline:**
   ```bash
   # Full pipeline
   ./run-local.sh

   # Specific step
   ./run-local.sh go-service build
   ./run-local.sh go-service test
   ```

### Option 2: Direct Command

```bash
# Install shipwright (if not already installed)
curl -L https://github.com/pablogore/shipwright/releases/latest/download/shipwright-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/') -o shipwright
chmod +x shipwright
sudo mv shipwright /usr/local/bin/

# Run pipeline locally
shipwright --pipeline go-service --local

# Run specific step
shipwright --pipeline go-service --step build --local
```

## ⚙️ Configuration

### Create `.shipwright.yml` in your project root:

```yaml
pipeline:
  name: go-service
  steps:
    - setup
    - build
    - test
    - lint
    - security
  coverage: 90
  verbose: true

environment: dev

git:
  ref: main
  protocol: https

security:
  enable_vuln_check: true
  enable_linting: true
```

**Note**: The `steps` field now uses a simple list format. Steps are executed in order.

## 📋 Available Steps

| Step | Description | Local Support |
|------|-------------|---------------|
| `setup` | Install dependencies (`go mod download`, `go mod tidy`) | ✅ Yes |
| `build` | Build the application (`go build`) | ✅ Yes |
| `test` | Run tests with coverage (`go test`) | ✅ Yes |
| `lint` | Run linters (`go vet`, `golangci-lint`) | ✅ Yes |
| `security` | Security scanning (`gosec`, `govulncheck`) | ✅ Yes |
| `tag` | Create git tags | ⚠️ Limited |
| `package` | Package artifacts | ⚠️ Limited |
| `push` | Push to registry | ❌ No (skipped in local mode) |
| `release` | Create releases | ❌ No (skipped in local mode) |

**Note**: Steps like `push` and `release` are automatically skipped in local mode since they require cloud services.

## 💡 Usage Examples

### Run Full Pipeline
```bash
./run-local.sh go-service
```

### Run Specific Step
```bash
./run-local.sh go-service build
./run-local.sh go-service test
./run-local.sh go-service lint
./run-local.sh go-service security
```

### With Custom Coverage Threshold
```bash
./run-local.sh go-service "" 95
```

### With Custom Config File
```bash
./run-local.sh go-service "" 90 .my-custom-config.yml
```

### Using Direct Commands

```bash
# Full pipeline
shipwright --pipeline go-service --local --config .shipwright.yml

# Only build step
shipwright --pipeline go-service --step build --local

# Only test step
shipwright --pipeline go-service --step test --local --coverage 95

# Build and test only
shipwright --pipeline go-service --only-build --local
shipwright --pipeline go-service --only-test --local
```

## 🔧 Requirements

**For `--local` mode (without Docker):**
- ✅ Go installed (version specified in `go.mod`)
- ✅ Internet access (for downloading dependencies)
- ✅ Optional but recommended:
  - `golangci-lint` for advanced linting
  - `gosec` for security scanning
  - `govulncheck` for vulnerability checking

**Installation of optional tools:**
```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Install govulncheck (Go 1.18+)
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## 🧪 Testing Your Pipeline Locally

### 1. Test Individual Steps

Test each step independently to isolate issues:

```bash
# Test setup
./run-local.sh go-service setup

# Test build
./run-local.sh go-service build

# Test with coverage
./run-local.sh go-service test
```

### 2. Test Full Pipeline

Run the complete pipeline as it would run in CI:

```bash
./run-local.sh go-service
```

### 3. Validate Configuration

Check that your configuration is valid:

```bash
shipwright --pipeline go-service --config .shipwright.yml --list-steps
```

### 4. Test Different Configurations

Create test configurations to validate different scenarios:

```bash
# Test with different coverage threshold
./run-local.sh go-service "" 80 .test-config.yml

# Test with minimal steps
./run-local.sh go-service "" 90 .minimal-config.yml
```

## 📊 Coverage Reports

When running tests locally, coverage reports are generated in the `coverage/` directory:

- `coverage/coverage.out` - Raw coverage data
- `coverage/coverage.html` - HTML coverage report (open in browser)

```bash
# View coverage report
open coverage/coverage.html  # macOS
xdg-open coverage/coverage.html  # Linux
```

## 🐛 Troubleshooting

### Issue: "shipwright not found"
**Solution**: Install shipwright (see Quick Start section)

### Issue: "Go is not installed"
**Solution**: Install Go from https://golang.org/dl/

### Issue: "not a Go project"
**Solution**: Make sure you're running from the project root where `go.mod` exists

### Issue: "golangci-lint not available"
**Solution**: This is optional. Install it or the lint step will use basic `go vet` only.

### Issue: Tests fail locally but pass in CI
**Check**:
- Go version matches CI environment
- Dependencies are up to date: `go mod download && go mod tidy`
- Environment variables are set correctly

### Issue: Coverage threshold not met
**Solution**: 
- Review coverage report: `open coverage/coverage.html`
- Increase test coverage or adjust threshold in config

## 🔄 Workflow Integration

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Run local pipeline before commit
./run-local.sh go-service test
```

### Makefile Integration

Add to your `Makefile`:

```makefile
.PHONY: test-local
test-local:
	./run-local.sh go-service test

.PHONY: build-local
build-local:
	./run-local.sh go-service build

.PHONY: pipeline-local
pipeline-local:
	./run-local.sh go-service
```

### CI/CD Validation

Before pushing to CI, validate locally:

```bash
# Run the same pipeline that CI will run
./run-local.sh go-service
```

## 📚 Advanced Usage

### Custom Step Execution

You can execute any supported step:

```bash
shipwright --pipeline go-service --step lint --local
shipwright --pipeline go-service --step security --local
```

### Verbose Output

Enable verbose logging for debugging:

```bash
shipwright --pipeline go-service --local --verbose
```

### Environment-Specific Configs

Use different configs for different environments:

```bash
# Development
shipwright --pipeline go-service --config .shipwright.dev.yml --local

# Staging
shipwright --pipeline go-service --config .shipwright.staging.yml --local
```

## 🎓 Best Practices

1. **Always test locally first**: Run pipelines locally before pushing to CI
2. **Use version control**: Commit your `.shipwright.yml` to version control
3. **Match CI environment**: Use the same Go version locally as in CI
4. **Keep dependencies updated**: Run `go mod tidy` regularly
5. **Test incrementally**: Test individual steps before running full pipeline

## 🆘 Getting Help

- **Documentation**: See main [README.md](../../README.md)
- **Issues**: [GitHub Issues](https://github.com/pablogore/shipwright/issues)
- **Discussions**: [GitHub Discussions](https://github.com/pablogore/shipwright/discussions)
