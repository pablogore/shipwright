# Syntegrity Dagger

A unified CI/CD pipeline library for Go projects, built on top of Dagger SDK. Syntegrity Dagger provides standardized, reusable pipelines that can be easily integrated into any Go project's CI/CD workflow.

## 🚀 Features

- **Unified Pipeline Architecture**: Standardized CI/CD pipelines for different Go project types
- **Dagger Integration**: Built on Dagger SDK for container-native pipeline execution
- **Multiple Pipeline Types**: Support for go-kit, docker-go, and infrastructure pipelines
- **Flexible Configuration**: YAML-based configuration with environment variable overrides
- **Extensible Design**: Plugin architecture for custom steps and hooks
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Security First**: Built-in vulnerability scanning and security checks

## 📋 Supported Pipeline Types

### Go-Kit Pipeline
Optimized for microservices built with the go-kit framework:
- Repository cloning (SSH/HTTPS)
- Dependency management
- Unit testing with coverage
- Linting and security scanning
- Docker image building
- Container registry publishing

### Docker-Go Pipeline
Standard pipeline for Go applications with Docker:
- Multi-stage Docker builds
- Cross-platform compilation
- Container optimization
- Registry publishing
- Health checks

### Infrastructure Pipeline
For infrastructure and deployment automation:
- Terraform validation
- Infrastructure testing
- Deployment automation
- Environment management

## 🛠️ Installation

### Quick Install

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/getsyntegrity/syntegrity-dagger/main/install.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/getsyntegrity/syntegrity-dagger/main/install.sh | bash -s -- -v v1.0.0
```

### Manual Installation

```bash
# Download binary for your platform
PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/')
VERSION="latest"  # or specific version like "v1.0.0"

curl -L "https://github.com/getsyntegrity/syntegrity-dagger/releases/download/${VERSION}/syntegrity-dagger-${PLATFORM}" -o syntegrity-dagger
chmod +x syntegrity-dagger
sudo mv syntegrity-dagger /usr/local/bin/
```

## 🚀 Quick Start

### Basic Usage

```bash
# Run go-kit pipeline
syntegrity-dagger --pipeline go-kit --env dev --coverage 90

# Run docker-go pipeline
syntegrity-dagger --pipeline docker-go --env prod

# Run infrastructure pipeline
syntegrity-dagger --pipeline infra --env staging
```

### Configuration File

Create a `.syntegrity-dagger.yml` file:

```yaml
pipeline:
  name: go-kit
  coverage: 90
  skip_push: false
  only_build: false
  only_test: false
  verbose: true

environment: dev

git:
  ref: main
  protocol: ssh

registry:
  url: registry.example.com
  username: ${REGISTRY_USERNAME}
  password: ${REGISTRY_PASSWORD}

steps:
  - name: setup
    required: true
    timeout: 5m
  - name: build
    required: true
    timeout: 10m
  - name: test
    required: true
    timeout: 15m
```

## 🔧 Command Line Interface

### Available Commands

```bash
# Show help
syntegrity-dagger --help

# Show version
syntegrity-dagger --version

# List available pipelines
syntegrity-dagger --list-pipelines

# List steps for a pipeline
syntegrity-dagger --list-steps --pipeline go-kit

# Execute specific step
syntegrity-dagger --pipeline go-kit --step build

# Execute with configuration file
syntegrity-dagger --config .syntegrity-dagger.yml
```

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--pipeline` | Pipeline type to execute | Required |
| `--env` | Environment (dev, staging, prod) | dev |
| `--coverage` | Minimum test coverage percentage | 80 |
| `--config` | Path to configuration file | - |
| `--step` | Execute specific step only | - |
| `--only-build` | Execute build step only | false |
| `--only-test` | Execute test step only | false |
| `--local` | Run locally without Docker | false |
| `--verbose` | Enable verbose logging | false |

## 🔄 CI/CD Integration

### GitHub Actions

```yaml
name: CI/CD Pipeline
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v5
    
    - name: Install Syntegrity Dagger
      run: |
        curl -fsSL https://raw.githubusercontent.com/getsyntegrity/syntegrity-dagger/main/install.sh | bash
    
    - name: Run Pipeline
      run: |
        syntegrity-dagger --pipeline go-kit --env dev --coverage 90
      env:
        REGISTRY_USERNAME: ${{ secrets.REGISTRY_USERNAME }}
        REGISTRY_PASSWORD: ${{ secrets.REGISTRY_PASSWORD }}
```

### GitLab CI

```yaml
stages:
  - build
  - test
  - deploy

variables:
  SYNTERGRITY_VERSION: "latest"

before_script:
  - curl -fsSL https://raw.githubusercontent.com/getsyntegrity/syntegrity-dagger/main/install.sh | bash -s -- -v $SYNTERGRITY_VERSION

build:
  stage: build
  script:
    - syntegrity-dagger --pipeline go-kit --only-build
```

### Jenkins

```groovy
pipeline {
    agent any
    
    stages {
        stage('Setup') {
            steps {
                sh '''
                    curl -fsSL https://raw.githubusercontent.com/getsyntegrity/syntegrity-dagger/main/install.sh | bash
                '''
            }
        }
        
        stage('Build') {
            steps {
                sh 'syntegrity-dagger --pipeline go-kit --only-build'
            }
        }
        
        stage('Test') {
            steps {
                sh 'syntegrity-dagger --pipeline go-kit --only-test --coverage 90'
            }
        }
    }
}
```

## 🏗️ Architecture

Syntegrity Dagger follows a modular architecture with clear separation of concerns:

- **Application Layer**: CLI interface and application lifecycle management
- **Pipeline Layer**: Pipeline implementations and registry
- **Step Layer**: Individual pipeline steps (build, test, lint, etc.)
- **Infrastructure Layer**: Dagger integration and container management
- **Configuration Layer**: Configuration management and validation

For detailed architecture documentation, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## 🔧 Development

### Prerequisites

- Go 1.25.1 or later
- Docker (for container-based pipelines)
- Make (for build automation)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/getsyntegrity/syntegrity-dagger.git
cd syntegrity-dagger

# Install dependencies
go mod download

# Build the binary
make build

# Run tests
make test

# Run linting
make lint
```

### Project Structure

```
syntegrity-dagger/
├── cmd/                    # CLI commands
├── internal/
│   ├── app/               # Application layer
│   ├── config/            # Configuration management
│   ├── interfaces/        # Interface definitions
│   └── pipelines/         # Pipeline implementations
├── examples/              # Usage examples
├── docs/                  # Documentation
└── tests/                 # Integration tests
```

## 📚 Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) - Detailed system architecture
- [Pipeline Development](docs/PIPELINE_DEVELOPMENT.md) - Creating custom pipelines
- [Configuration Reference](docs/CONFIGURATION.md) - Configuration options
- [API Reference](docs/API.md) - Programmatic API documentation
- [Examples](examples/) - Usage examples and templates

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## 🆘 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/getsyntegrity/syntegrity-dagger/issues)
- **Discussions**: [GitHub Discussions](https://github.com/getsyntegrity/syntegrity-dagger/discussions)

## 🗺️ Roadmap

- [ ] Support for additional programming languages
- [ ] Kubernetes deployment pipelines
- [ ] Advanced security scanning
- [ ] Pipeline visualization and monitoring
- [ ] Plugin marketplace

---

**Syntegrity Dagger** - Unified CI/CD pipelines for modern Go applications.
