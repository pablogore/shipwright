# Configuration Reference

This document provides a comprehensive reference for configuring Syntegrity Dagger pipelines, including all available options, environment variables, and configuration file formats.

## Configuration Hierarchy

Syntegrity Dagger uses a layered configuration system with the following precedence (highest to lowest):

1. **Command Line Flags** - Override all other settings
2. **Environment Variables** - Override file and default settings
3. **Configuration Files** - Override default settings
4. **Default Values** - Built-in defaults

```mermaid
flowchart TD
    A[Command Line Flags] --> B[Environment Variables]
    B --> C[Configuration Files]
    C --> D[Default Values]
    
    A1[--pipeline go-kit] --> A
    A2[--env prod] --> A
    A3[--coverage 90] --> A
    
    B1[SYNTERGRITY_PIPELINE] --> B
    B2[SYNTERGRITY_ENV] --> B
    B3[SYNTERGRITY_COVERAGE] --> B
    
    C1[.syntegrity-dagger.yml] --> C
    C2[config/production.yml] --> C
    
    D1[Pipeline: go-kit] --> D
    D2[Environment: dev] --> D
    D3[Coverage: 80] --> D
```

## Configuration File Format

### YAML Configuration

Create a `.syntegrity-dagger.yml` file in your project root:

```yaml
# Pipeline Configuration
pipeline:
  name: go-kit                    # Pipeline type (go-kit, docker-go, infra)
  coverage: 90                    # Minimum test coverage percentage
  skip_push: false               # Skip pushing to registry
  only_build: false              # Execute build step only
  only_test: false               # Execute test step only
  verbose: true                  # Enable verbose logging
  timeout: 30m                   # Global pipeline timeout

# Environment Configuration
environment: dev                 # Environment (dev, staging, prod)

# Git Configuration
git:
  url: https://github.com/user/repo.git
  ref: main                      # Git reference (branch, tag, commit)
  protocol: ssh                  # Git protocol (ssh, https)
  credentials:
    username: ${GIT_USERNAME}    # Git username
    password: ${GIT_PASSWORD}    # Git password/token
    ssh_key: ${SSH_PRIVATE_KEY}  # SSH private key

# Container Registry Configuration
registry:
  url: registry.example.com      # Registry URL
  username: ${REGISTRY_USERNAME} # Registry username
  password: ${REGISTRY_PASSWORD} # Registry password
  namespace: myorg               # Registry namespace
  insecure: false               # Allow insecure registry

# Docker Configuration
docker:
  build_args:                    # Docker build arguments
    - BUILD_ENV=production
    - VERSION=${VERSION}
  labels:                        # Docker labels
    - "org.opencontainers.image.source=https://github.com/user/repo"
    - "org.opencontainers.image.version=${VERSION}"
  platforms:                     # Target platforms
    - linux/amd64
    - linux/arm64

# Build Configuration
build:
  go_version: 1.25.1            # Go version to use
  cgo_enabled: false            # Enable CGO
  build_flags:                  # Additional build flags
    - -ldflags=-s -w
    - -trimpath
  test_flags:                   # Additional test flags
    - -race
    - -coverprofile=coverage.out

# Security Configuration
security:
  enable_scanning: true         # Enable vulnerability scanning
  scan_level: medium            # Scan level (low, medium, high, critical)
  fail_on_vulnerabilities: true # Fail pipeline on vulnerabilities
  exclude_packages:             # Packages to exclude from scanning
    - github.com/example/package

# Linting Configuration
linting:
  enabled: true                 # Enable linting
  level: medium                 # Lint level (low, medium, high)
  fail_on_issues: true          # Fail pipeline on lint issues
  exclude_rules:                # Lint rules to exclude
    - gocyclo
    - gomnd

# Step Configuration
steps:
  - name: setup                 # Step name
    enabled: true               # Enable/disable step
    required: true              # Step is required
    timeout: 5m                 # Step timeout
    retries: 3                  # Number of retries
    parallel: false             # Allow parallel execution
    
  - name: build
    enabled: true
    required: true
    timeout: 10m
    retries: 2
    parallel: false
    
  - name: test
    enabled: true
    required: true
    timeout: 15m
    retries: 1
    parallel: true              # Tests can run in parallel

# Hook Configuration
hooks:
  pre_pipeline:                 # Pre-pipeline hooks
    - name: notify_start
      command: "curl -X POST ${WEBHOOK_URL}/start"
      
  post_pipeline:                # Post-pipeline hooks
    - name: notify_complete
      command: "curl -X POST ${WEBHOOK_URL}/complete"
      
  pre_step:                     # Pre-step hooks
    - name: backup_artifacts
      command: "tar -czf backup-${TIMESTAMP}.tar.gz ./artifacts"
      
  post_step:                    # Post-step hooks
    - name: cleanup_temp
      command: "rm -rf ./temp"

# Notification Configuration
notifications:
  webhooks:                     # Webhook notifications
    - url: ${SLACK_WEBHOOK_URL}
      events: [success, failure]
      template: |
        Pipeline {{.Pipeline}} {{.Status}} for {{.Environment}}
        
  email:                        # Email notifications
    smtp_server: smtp.gmail.com
    smtp_port: 587
    username: ${EMAIL_USERNAME}
    password: ${EMAIL_PASSWORD}
    recipients:
      - devops@example.com
      - team@example.com

# Cache Configuration
cache:
  enabled: true                 # Enable caching
  ttl: 24h                      # Cache time-to-live
  paths:                        # Paths to cache
    - ./vendor
    - ./node_modules
    - ~/.cache/go-build
```

## Environment Variables

### Core Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_PIPELINE` | Pipeline type to execute | `go-kit` | `docker-go` |
| `SYNTERGRITY_ENV` | Environment name | `dev` | `production` |
| `SYNTERGRITY_COVERAGE` | Minimum test coverage | `80` | `90` |
| `SYNTERGRITY_VERBOSE` | Enable verbose logging | `false` | `true` |
| `SYNTERGRITY_CONFIG` | Configuration file path | `.syntegrity-dagger.yml` | `config/prod.yml` |

### Git Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_GIT_URL` | Git repository URL | - | `https://github.com/user/repo.git` |
| `SYNTERGRITY_GIT_REF` | Git reference | `main` | `develop` |
| `SYNTERGRITY_GIT_PROTOCOL` | Git protocol | `https` | `ssh` |
| `SYNTERGRITY_GIT_USERNAME` | Git username | - | `git` |
| `SYNTERGRITY_GIT_PASSWORD` | Git password/token | - | `ghp_xxx` |
| `SYNTERGRITY_SSH_PRIVATE_KEY` | SSH private key | - | `-----BEGIN OPENSSH PRIVATE KEY-----` |

### Registry Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_REGISTRY_URL` | Container registry URL | - | `registry.example.com` |
| `SYNTERGRITY_REGISTRY_USERNAME` | Registry username | - | `myuser` |
| `SYNTERGRITY_REGISTRY_PASSWORD` | Registry password | - | `mypassword` |
| `SYNTERGRITY_REGISTRY_NAMESPACE` | Registry namespace | - | `myorg` |
| `SYNTERGRITY_REGISTRY_INSECURE` | Allow insecure registry | `false` | `true` |

### Build Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_GO_VERSION` | Go version to use | `1.25.1` | `1.24.0` |
| `SYNTERGRITY_CGO_ENABLED` | Enable CGO | `false` | `true` |
| `SYNTERGRITY_BUILD_FLAGS` | Additional build flags | - | `-ldflags=-s -w` |
| `SYNTERGRITY_TEST_FLAGS` | Additional test flags | - | `-race -cover` |

### Security Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_SECURITY_ENABLED` | Enable security scanning | `true` | `false` |
| `SYNTERGRITY_SECURITY_LEVEL` | Security scan level | `medium` | `high` |
| `SYNTERGRITY_FAIL_ON_VULNERABILITIES` | Fail on vulnerabilities | `true` | `false` |

### Notification Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SYNTERGRITY_WEBHOOK_URL` | Webhook URL for notifications | - | `https://hooks.slack.com/xxx` |
| `SYNTERGRITY_EMAIL_SMTP_SERVER` | SMTP server for email | - | `smtp.gmail.com` |
| `SYNTERGRITY_EMAIL_USERNAME` | Email username | - | `user@gmail.com` |
| `SYNTERGRITY_EMAIL_PASSWORD` | Email password | - | `password` |

## Command Line Options

### Global Options

```bash
syntegrity-dagger [OPTIONS] [COMMAND]
```

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--help, -h` | Show help information | - | `--help` |
| `--version, -v` | Show version information | - | `--version` |
| `--config, -c` | Configuration file path | `.syntegrity-dagger.yml` | `--config prod.yml` |
| `--verbose` | Enable verbose logging | `false` | `--verbose` |
| `--dry-run` | Show what would be executed | `false` | `--dry-run` |

### Pipeline Options

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--pipeline, -p` | Pipeline type to execute | Required | `--pipeline go-kit` |
| `--env, -e` | Environment name | `dev` | `--env production` |
| `--coverage` | Minimum test coverage | `80` | `--coverage 90` |
| `--timeout` | Pipeline timeout | `30m` | `--timeout 1h` |

### Execution Options

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--step` | Execute specific step only | - | `--step build` |
| `--only-build` | Execute build step only | `false` | `--only-build` |
| `--only-test` | Execute test step only | `false` | `--only-test` |
| `--local` | Run locally without containers | `false` | `--local` |
| `--skip-push` | Skip pushing to registry | `false` | `--skip-push` |

### Information Options

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--list-pipelines` | List available pipelines | - | `--list-pipelines` |
| `--list-steps` | List pipeline steps | - | `--list-steps --pipeline go-kit` |
| `--show-config` | Show resolved configuration | - | `--show-config` |

## Pipeline-Specific Configuration

### Go-Kit Pipeline

```yaml
pipeline:
  name: go-kit
  
# Go-Kit specific options
go_kit:
  service_name: my-service
  service_version: 1.0.0
  endpoints:
    - name: health
      path: /health
      method: GET
    - name: metrics
      path: /metrics
      method: GET
  middleware:
    - logging
    - metrics
    - tracing
  transport:
    http:
      port: 8080
      host: 0.0.0.0
    grpc:
      port: 9090
      host: 0.0.0.0
```

### Docker-Go Pipeline

```yaml
pipeline:
  name: docker-go
  
# Docker-Go specific options
docker_go:
  base_image: golang:1.25.1-alpine
  final_image: alpine:3.18
  build_stages:
    - name: builder
      commands:
        - go mod download
        - go build -o app ./cmd/app
    - name: runtime
      commands:
        - cp /builder/app /app/
        - chmod +x /app/app
  ports:
    - 8080
  health_check:
    command: ["/app/app", "health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

### Infrastructure Pipeline

```yaml
pipeline:
  name: infra
  
# Infrastructure specific options
infrastructure:
  provider: aws
  region: us-west-2
  terraform:
    version: 1.5.0
    backend:
      type: s3
      bucket: my-terraform-state
      key: infrastructure/terraform.tfstate
    variables:
      - name: environment
        value: ${SYNTERGRITY_ENV}
      - name: region
        value: us-west-2
  kubernetes:
    version: 1.28
    namespace: ${SYNTERGRITY_ENV}
    resources:
      - name: deployment
        file: k8s/deployment.yaml
      - name: service
        file: k8s/service.yaml
      - name: ingress
        file: k8s/ingress.yaml
```

## Configuration Validation

### Required Fields

The following fields are required for all pipelines:

- `pipeline.name` - Pipeline type
- `git.url` - Git repository URL
- `registry.url` - Container registry URL

### Validation Rules

```yaml
# Example validation configuration
validation:
  required_fields:
    - pipeline.name
    - git.url
    - registry.url
    
  rules:
    - field: pipeline.coverage
      type: number
      min: 0
      max: 100
      
    - field: pipeline.timeout
      type: duration
      min: 1m
      max: 2h
      
    - field: git.protocol
      type: enum
      values: [ssh, https]
      
    - field: registry.url
      type: url
      schemes: [http, https]
```

## Configuration Examples

### Development Environment

```yaml
# .syntegrity-dagger.yml
pipeline:
  name: go-kit
  coverage: 80
  verbose: true

environment: dev

git:
  url: https://github.com/myorg/myservice.git
  ref: develop
  protocol: https

registry:
  url: localhost:5000
  namespace: dev
  insecure: true

build:
  go_version: 1.25.1
  test_flags:
    - -race
    - -short

security:
  enable_scanning: false  # Skip security scans in dev

notifications:
  webhooks:
    - url: ${SLACK_DEV_WEBHOOK}
      events: [failure]  # Only notify on failures in dev
```

### Production Environment

```yaml
# config/production.yml
pipeline:
  name: go-kit
  coverage: 95
  verbose: false
  timeout: 1h

environment: production

git:
  url: https://github.com/myorg/myservice.git
  ref: main
  protocol: ssh

registry:
  url: registry.example.com
  namespace: production
  insecure: false

build:
  go_version: 1.25.1
  build_flags:
    - -ldflags=-s -w
    - -trimpath

security:
  enable_scanning: true
  scan_level: high
  fail_on_vulnerabilities: true

notifications:
  webhooks:
    - url: ${SLACK_PROD_WEBHOOK}
      events: [success, failure]
  email:
    smtp_server: smtp.gmail.com
    smtp_port: 587
    username: ${EMAIL_USERNAME}
    password: ${EMAIL_PASSWORD}
    recipients:
      - devops@example.com
      - oncall@example.com
```

### Multi-Environment Configuration

```yaml
# config/base.yml
pipeline:
  name: go-kit
  coverage: 90

git:
  url: https://github.com/myorg/myservice.git
  protocol: ssh

registry:
  url: registry.example.com
  namespace: ${SYNTERGRITY_ENV}

build:
  go_version: 1.25.1

security:
  enable_scanning: true
  scan_level: medium
```

```yaml
# config/development.yml
extends: base.yml

pipeline:
  coverage: 80
  verbose: true

environment: dev

registry:
  namespace: dev
  insecure: true

security:
  scan_level: low
  fail_on_vulnerabilities: false
```

```yaml
# config/production.yml
extends: base.yml

pipeline:
  coverage: 95
  verbose: false

environment: production

registry:
  namespace: production
  insecure: false

security:
  scan_level: high
  fail_on_vulnerabilities: true
```

## Troubleshooting Configuration

### Common Issues

1. **Configuration not found**: Ensure the config file exists and is readable
2. **Invalid YAML**: Check YAML syntax and indentation
3. **Missing required fields**: Verify all required configuration is present
4. **Environment variable not set**: Check that all referenced environment variables are defined
5. **Invalid values**: Validate configuration values against expected types and ranges

### Debugging Configuration

```bash
# Show resolved configuration
syntegrity-dagger --show-config --pipeline go-kit

# Validate configuration without execution
syntegrity-dagger --dry-run --pipeline go-kit

# Show configuration with verbose logging
syntegrity-dagger --verbose --pipeline go-kit
```

### Configuration Testing

```bash
# Test configuration loading
syntegrity-dagger --config test-config.yml --dry-run

# Validate specific environment
SYNTERGRITY_ENV=production syntegrity-dagger --show-config

# Test with different configuration files
syntegrity-dagger --config config/dev.yml --pipeline go-kit --dry-run
```
