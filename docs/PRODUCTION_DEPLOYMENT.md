# Production Deployment Guide

This guide provides step-by-step instructions for deploying Shipwright in production environments.

## Prerequisites

1. **Dagger Engine**: Ensure Dagger engine is installed and running
   ```bash
   dagger version
   # If not installed: https://docs.dagger.io/install
   ```

2. **Go Environment**: Go 1.21 or later
   ```bash
   go version
   ```

3. **Docker Registry Access**: Valid credentials for your container registry
   - GitHub Container Registry (ghcr.io)
   - Docker Hub
   - Private registries

4. **Git Access**: SSH keys or HTTPS tokens for repository access

## Configuration

### 1. Create Production Configuration

Copy the production template:

```bash
cp examples/configs/production.yml .shipwright.yml
```

### 2. Configure Registry

Edit `.shipwright.yml`:

```yaml
registry:
  baseUrl: "https://ghcr.io"
  image: "your-org/your-service"
```

Set credentials via environment variables:

```bash
export REGISTRY_USER="your-username"
export REGISTRY_TOKEN="your-token"
```

Or use GitHub Actions secrets:

```yaml
env:
  REGISTRY_USER: ${{ secrets.GITHUB_TOKEN }}
  REGISTRY_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 3. Configure Pipeline Steps

Adjust timeouts and retries based on your service requirements:

```yaml
stepConfigs:
  - name: build
    timeout: "15m"  # Increase for large projects
    retries: 1
  - name: test
    timeout: "20m"  # Increase for extensive test suites
    retries: 2
```

**Important**: Maximum timeout is 2 hours. Exceeding this will be capped automatically.

### 4. Set Minimum Coverage

For production, coverage must be at least 80%:

```yaml
pipeline:
  coverage: 90.0  # Recommended: 90%+
```

## Security Best Practices

### 1. Secrets Management

**Never commit secrets to version control!**

- Use environment variables for all sensitive data
- Use secret management systems (HashiCorp Vault, AWS Secrets Manager, etc.)
- Rotate credentials regularly

### 2. Registry Authentication

```bash
# GitHub Container Registry
export REGISTRY_USER="x-access-token"
export REGISTRY_TOKEN="${GITHUB_TOKEN}"

# Docker Hub
export REGISTRY_USER="your-dockerhub-username"
export REGISTRY_TOKEN="${DOCKERHUB_TOKEN}"
```

### 3. Git Authentication

For CI/CD environments:

```bash
# HTTPS with token
export GITHUB_TOKEN="${GITHUB_TOKEN}"

# SSH (ensure SSH_PRIVATE_KEY is set)
export SSH_PRIVATE_KEY="${SSH_PRIVATE_KEY}"
```

### 4. Network Security

- Use HTTPS for all registry and Git URLs
- Verify SSL certificates
- Use private registries when possible
- Implement network policies to restrict access

## Deployment Steps

### 1. Validate Configuration

```bash
# Run health checks
shipwright --health

# Expected output:
# ✅ Dagger engine is accessible
# ✅ Registry is accessible: https://ghcr.io
# ✅ Git repository is accessible: https://github.com/...
```

### 2. Test Pipeline Locally

```bash
# Run pipeline locally first
shipwright --pipeline go-service --local

# Or test specific step
shipwright --pipeline go-service --step test --local
```

### 3. Execute Full Pipeline

```bash
# Run complete pipeline
shipwright --pipeline go-service

# With specific configuration
shipwright --pipeline go-service --config .shipwright.yml
```

## CI/CD Integration

### GitHub Actions

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Dagger
        uses: dagger/dagger-for-github@v5
      
      - name: Run Pipeline
        env:
          REGISTRY_USER: ${{ secrets.GITHUB_TOKEN }}
          REGISTRY_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          shipwright --pipeline go-service --config .shipwright.yml
```

### GitLab CI

```yaml
build:
  image: golang:1.25
  before_script:
    - curl -L https://dagger.io/dagger/install.sh | sh
    - export PATH="$PATH:/usr/local/bin"
  script:
    - shipwright --pipeline go-service --config .shipwright.yml
  variables:
    REGISTRY_USER: $CI_REGISTRY_USER
    REGISTRY_TOKEN: $CI_JOB_TOKEN
```

## Performance Tuning

### 1. Timeout Configuration

Adjust timeouts based on your service size:

- **Small services** (< 10k LOC): 5-10 minutes per step
- **Medium services** (10k-50k LOC): 10-20 minutes per step
- **Large services** (> 50k LOC): 20-30 minutes per step

### 2. Retry Strategy

Configure retries for network-dependent operations:

```yaml
stepConfigs:
  - name: push
    timeout: "10m"
    retries: 3  # More retries for network operations
  - name: test
    timeout: "20m"
    retries: 2  # Fewer retries for deterministic operations
```

### 3. Parallel Execution

Some steps can run in parallel (configure in your CI/CD system):

```yaml
# Example: Run lint and security checks in parallel
steps:
  - setup
  - build
  - test
  - [lint, security]  # Parallel execution
  - package
```

## Monitoring and Observability

### 1. Health Checks

Run health checks regularly:

```bash
# Scheduled health check
shipwright --health
```

### 2. Logging

Configure log levels:

```yaml
logging:
  level: info  # Options: debug, info, warn, error
```

### 3. Error Tracking

Monitor pipeline failures:

- Set up alerts for pipeline failures
- Track error rates and patterns
- Review logs regularly

## Troubleshooting

### Common Issues

1. **Dagger Engine Not Running**
   ```bash
   # Solution: Start Dagger engine
   dagger run echo test
   ```

2. **Registry Authentication Failed**
   ```bash
   # Solution: Verify credentials
   echo $REGISTRY_TOKEN
   shipwright --health
   ```

3. **Timeout Errors**
   ```bash
   # Solution: Increase timeout in configuration
   # Edit .shipwright.yml and increase step timeout
   ```

4. **Coverage Below Threshold**
   ```bash
   # Solution: Improve test coverage
   # Minimum: 80% for production
   # Recommended: 90%+
   ```

## Rollback Strategy

If a deployment fails:

1. **Identify the failing step**
   ```bash
   shipwright --pipeline go-service --step <step-name>
   ```

2. **Check logs** for detailed error messages

3. **Fix the issue** and re-run the pipeline

4. **Use previous image tag** if needed

## Support

For issues or questions:

1. Check the troubleshooting guide: `docs/TROUBLESHOOTING.md`
2. Review error messages (they include suggestions)
3. Run health checks: `shipwright --health`
4. Check Dagger documentation: https://docs.dagger.io

## Additional Resources

- [Security Best Practices](SECURITY_BEST_PRACTICES.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Configuration Reference](../examples/configs/production.yml)


