# Syntegrity Dagger Makefile
# Usage: make [target] [options]

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=syntegrity-dagger

# Tools
GOLANGCI_LINT=golangci-lint
GORELEASER=goreleaser

# Coverage threshold
COVERAGE_THRESHOLD=90
COVERAGE_THRESHOLD_100=100

# Default target
.DEFAULT_GOAL := help

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
WHITE := \033[1;37m
NC := \033[0m # No Color

.PHONY: all build clean test deps lint lint-fix tools-install release release-snapshot release-dry-run help coverage coverage-html coverage-report coverage-package coverage-file coverage-summary coverage-threshold coverage-100 local-run pipeline-local

# Help target
.PHONY: help
help: ## Show this help message
	@echo -e "$(BLUE)🚀 Syntegrity Dagger - Makefile$(NC)"
	@echo "=================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make all                # Complete pipeline: test, build, lint"
	@echo "  make build              # Build the application"
	@echo "  make test               # Run all tests"
	@echo "  make coverage           # Generate complete coverage report"
	@echo "  make lint               # Run linter"
	@echo "  make local-run          # Run pipeline locally"
	@echo "  make tools-install      # Install development tools"
	@echo ""
	@echo "Coverage:"
	@echo "  make coverage           # Generate comprehensive ASCII coverage report with threshold validation"
	@echo "  make coverage-100       # Generate 100% coverage report including all packages"
	@echo "  make coverage-html      # Generate HTML coverage report"
	@echo "  make coverage-report    # Generate detailed coverage reports by package and file"
	@echo "  make coverage-package   # Generate detailed ASCII coverage report by package"
	@echo "  make coverage-file      # Generate detailed ASCII coverage report by file"
	@echo "  make coverage-summary   # Generate comprehensive coverage summary with statistics"
	@echo "  make coverage-threshold # Validate coverage against threshold with detailed reporting"
	@echo ""
	@echo "Development:"
	@echo "  make fmt                # Format code"
	@echo "  make vet                # Run go vet"
	@echo "  make quality            # Run all quality checks"
	@echo "  make clean              # Clean build artifacts"
	@echo ""
	@echo "Pipeline:"
	@echo "  make pipeline-local     # Run complete pipeline locally"
	@echo "  make pipeline-setup     # Run setup step locally"
	@echo "  make pipeline-build     # Run build step locally"
	@echo "  make pipeline-test      # Run test step locally"

all: test build

build: ## Build the application
	@echo -e "$(BLUE)Building application...$(NC)"
	$(GOBUILD) -o $(BINARY_NAME) .
	@echo -e "$(GREEN)✅ Build completed$(NC)"

clean: ## Clean build artifacts
	@echo -e "$(BLUE)Cleaning build artifacts...$(NC)"
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html
	rm -rf coverage/
	rm -rf logs/
	@echo -e "$(GREEN)✅ Clean completed$(NC)"

test: ## Run all tests
	@echo -e "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v -race ./...
	@echo -e "$(GREEN)✅ Tests completed$(NC)"

deps: ## Download and tidy dependencies
	@echo -e "$(BLUE)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo -e "$(GREEN)✅ Dependencies updated$(NC)"

# Install development tools
tools-install: ## Install development tools (golangci-lint, goreleaser)
	@echo -e "$(BLUE)Installing development tools...$(NC)"
	@echo -e "$(YELLOW)Installing golangci-lint via install script...$(NC)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.4.0
	@echo -e "$(YELLOW)Verifying golangci-lint installation...$(NC)"
	@golangci-lint --version
	@echo -e "$(YELLOW)Installing goreleaser...$(NC)"
	@$(GOGET) github.com/goreleaser/goreleaser@latest
	@echo -e "$(GREEN)✅ Development tools installed$(NC)"

# Run golangci-lint
lint: ## Run linter
	@echo -e "$(BLUE)Running linter...$(NC)"
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run --timeout=5m; \
	else \
		echo -e "$(YELLOW)Warning: golangci-lint not installed$(NC)"; \
		echo "Install with: make tools-install"; \
	fi
	@echo -e "$(GREEN)✅ Linting completed$(NC)"

# Run golangci-lint with auto-fix
lint-fix: ## Run linter with auto-fix
	@echo -e "$(BLUE)Running linter with auto-fix...$(NC)"
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run --fix --timeout=5m; \
	else \
		echo -e "$(YELLOW)Warning: golangci-lint not installed$(NC)"; \
		echo "Install with: make tools-install"; \
	fi
	@echo -e "$(GREEN)✅ Linting with auto-fix completed$(NC)"

# Check if golangci-lint is installed
lint-check: ## Check if golangci-lint is installed
	@echo -e "$(BLUE)Checking golangci-lint installation...$(NC)"
	@which $(GOLANGCI_LINT) > /dev/null || (echo -e "$(RED)golangci-lint not found. Run 'make tools-install' to install it.$(NC)" && exit 1)
	@echo -e "$(GREEN)✅ golangci-lint is installed$(NC)"

# Run all checks (lint + test)
check: lint-check lint test ## Run all checks (lint + test)

# Format code
fmt: ## Format code with gofmt
	@echo -e "$(BLUE)Formatting code...$(NC)"
	$(GOCMD) fmt ./...
	@echo -e "$(GREEN)✅ Code formatting completed$(NC)"

# Run go vet
vet: ## Run go vet
	@echo -e "$(BLUE)Running go vet...$(NC)"
	$(GOCMD) vet ./...
	@echo -e "$(GREEN)✅ Go vet completed$(NC)"

# Security check
security: ## Run security vulnerability check
	@echo -e "$(BLUE)Running security vulnerability check...$(NC)"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./... | tee vuln_report.txt
	@if grep -q "Vulnerabilities found" vuln_report.txt; then \
		echo -e "$(RED)❌ Security vulnerabilities detected! Please update dependencies.$(NC)"; \
		cat vuln_report.txt; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ No security vulnerabilities found.$(NC)"; \
	fi
	@rm -f vuln_report.txt

# Run all code quality checks
quality: fmt vet lint test security ## Run all code quality checks (fmt, vet, lint, test, security)

# Coverage targets
coverage: ## Generate comprehensive ASCII coverage report with threshold validation
	@echo -e "$(BLUE)Generating comprehensive coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                           📊 COVERAGE SUMMARY                                ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total Coverage: $${COVERAGE}%$(NC)"; \
	if [ -n "$$COVERAGE" ] && [ "$$COVERAGE" != "" ]; then \
		if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l 2>/dev/null || echo "1") -eq 1 ]; then \
			echo -e "$(RED)❌ Coverage $${COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
			exit 1; \
		else \
			echo -e "$(GREEN)✅ Coverage meets threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		fi; \
	else \
		echo -e "$(YELLOW)⚠️  Could not determine coverage percentage$(NC)"; \
	fi

coverage-html: ## Generate HTML coverage report
	@echo -e "$(BLUE)Generating HTML coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo -e "$(GREEN)✅ HTML coverage report generated: coverage/coverage.html$(NC)"

coverage-report: coverage-package coverage-file ## Generate detailed coverage reports by package and file

coverage-package: ## Generate detailed ASCII coverage report by package
	@echo -e "$(BLUE)Generating package coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        📦 PACKAGE COVERAGE REPORT                            ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)Package Coverage Breakdown:$(NC)"
	@echo -e "$(WHITE)────────────────────────────$(NC)"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/app/" | grep -v "/config/" | \
	awk '{ \
		coverage = $$3; \
		gsub(/%/, "", coverage); \
		if (coverage >= 90) color = "$(GREEN)"; \
		else if (coverage >= 80) color = "$(YELLOW)"; \
		else if (coverage >= 70) color = "$(PURPLE)"; \
		else color = "$(RED)"; \
		printf "%s%-60s %s%6s%s\n", color, $$1, color, $$3, "$(NC)"; \
	}' | sort -k2 -nr
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total Package Coverage: $${COVERAGE}%$(NC)"

coverage-file: ## Generate detailed ASCII coverage report by file
	@echo -e "$(BLUE)Generating file coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                         📄 FILE COVERAGE REPORT                              ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)File Coverage Breakdown:$(NC)"
	@echo -e "$(WHITE)─────────────────────────$(NC)"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "\.go:" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | \
	awk '{ \
		coverage = $$3; \
		gsub(/%/, "", coverage); \
		if (coverage >= 90) color = "$(GREEN)"; \
		else if (coverage >= 80) color = "$(YELLOW)"; \
		else if (coverage >= 70) color = "$(PURPLE)"; \
		else color = "$(RED)"; \
		printf "%s%-70s %s%6s%s\n", color, $$1, color, $$3, "$(NC)"; \
	}' | sort -k2 -nr
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total File Coverage: $${COVERAGE}%$(NC)"

coverage-summary: ## Generate comprehensive coverage summary with statistics
	@echo -e "$(BLUE)Generating comprehensive coverage summary...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        📈 COMPREHENSIVE COVERAGE SUMMARY                     ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)Coverage Statistics:$(NC)"
	@echo -e "$(WHITE)────────────────────$(NC)"
	@TOTAL_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	PACKAGES=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | wc -l); \
	FILES=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "\.go:" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | wc -l); \
	HIGH_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | awk '{gsub(/%/, "", $$3); if ($$3 >= 90) print $$1}' | wc -l); \
	MEDIUM_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | awk '{gsub(/%/, "", $$3); if ($$3 >= 80 && $$3 < 90) print $$1}' | wc -l); \
	LOW_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | awk '{gsub(/%/, "", $$3); if ($$3 < 80) print $$1}' | wc -l); \
	echo -e "$(GREEN)Total Coverage: $${TOTAL_COVERAGE}%$(NC)"; \
	echo -e "$(BLUE)Total Packages: $${PACKAGES}$(NC)"; \
	echo -e "$(BLUE)Total Files: $${FILES}$(NC)"; \
	echo -e "$(GREEN)High Coverage (≥90%): $${HIGH_COVERAGE} packages$(NC)"; \
	echo -e "$(YELLOW)Medium Coverage (80-89%): $${MEDIUM_COVERAGE} packages$(NC)"; \
	echo -e "$(RED)Low Coverage (<80%): $${LOW_COVERAGE} packages$(NC)"; \
	echo ""; \
	if [ $$(echo "$$TOTAL_COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo -e "$(RED)❌ Coverage $${TOTAL_COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ Coverage meets threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
	fi

coverage-threshold: ## Validate coverage against threshold with detailed reporting
	@echo -e "$(BLUE)Validating coverage threshold...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        🎯 COVERAGE THRESHOLD VALIDATION                      ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@TOTAL_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(WHITE)Threshold Validation:$(NC)"; \
	echo -e "$(WHITE)─────────────────────$(NC)"; \
	echo -e "$(BLUE)Required Threshold: $(COVERAGE_THRESHOLD)%$(NC)"; \
	echo -e "$(BLUE)Current Coverage: $${TOTAL_COVERAGE}%$(NC)"; \
	echo ""; \
	if [ $$(echo "$$TOTAL_COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo -e "$(RED)❌ FAILED: Coverage $${TOTAL_COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		echo ""; \
		echo -e "$(YELLOW)Packages below threshold:$(NC)"; \
		$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | \
		awk -v threshold=$(COVERAGE_THRESHOLD) '{ \
			coverage = $$3; \
			gsub(/%/, "", coverage); \
			if (coverage < threshold) printf "$(RED)%-60s %6s$(NC)\n", $$1, $$3; \
		}'; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ PASSED: Coverage meets threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
	fi

# Local pipeline execution
local-run: ## Run pipeline locally
	@echo -e "$(BLUE)Running pipeline locally...$(NC)"
	@./$(BINARY_NAME) --local
	@echo -e "$(GREEN)✅ Local pipeline completed$(NC)"

pipeline-local: build local-run ## Build and run pipeline locally

pipeline-setup: build ## Run setup step locally
	@echo -e "$(BLUE)Running setup step locally...$(NC)"
	@./$(BINARY_NAME) --local --step setup
	@echo -e "$(GREEN)✅ Setup step completed$(NC)"

pipeline-build: build ## Run build step locally
	@echo -e "$(BLUE)Running build step locally...$(NC)"
	@./$(BINARY_NAME) --local --step build
	@echo -e "$(GREEN)✅ Build step completed$(NC)"

pipeline-test: build ## Run test step locally
	@echo -e "$(BLUE)Running test step locally...$(NC)"
	@./$(BINARY_NAME) --local --step test
	@echo -e "$(GREEN)✅ Test step completed$(NC)"

pipeline-lint: build ## Run lint step locally
	@echo -e "$(BLUE)Running lint step locally...$(NC)"
	@./$(BINARY_NAME) --local --step lint
	@echo -e "$(GREEN)✅ Lint step completed$(NC)"

pipeline-security: build ## Run security step locally
	@echo -e "$(BLUE)Running security step locally...$(NC)"
	@./$(BINARY_NAME) --local --step security
	@echo -e "$(GREEN)✅ Security step completed$(NC)"

# Log analysis and reporting
logs-analyze: ## Analyze pipeline logs and generate ASCII report
	@echo -e "$(BLUE)Analyzing pipeline logs...$(NC)"
	@mkdir -p logs
	@if [ -f scripts/log-analyzer.sh ]; then \
		./scripts/log-analyzer.sh; \
	else \
		echo -e "$(YELLOW)⚠️  log-analyzer.sh not found, creating basic log analysis$(NC)"; \
		echo -e "$(CYAN)📋 Pipeline Execution Summary$(NC)"; \
		echo "================================"; \
		echo "Last execution: $$(date)"; \
		echo "Status: Check logs/ directory for detailed information"; \
	fi

# GoReleaser targets
release: tools-install ## Create release with goreleaser
	@echo -e "$(BLUE)Creating release...$(NC)"
	$(GORELEASER) release --clean
	@echo -e "$(GREEN)✅ Release created$(NC)"

release-snapshot: tools-install ## Create snapshot release
	@echo -e "$(BLUE)Creating snapshot release...$(NC)"
	$(GORELEASER) release --snapshot --clean
	@echo -e "$(GREEN)✅ Snapshot release created$(NC)"

release-dry-run: tools-install ## Run dry-run release
	@echo -e "$(BLUE)Running dry-run release...$(NC)"
	$(GORELEASER) release --snapshot --skip-publish --clean
	@echo -e "$(GREEN)✅ Dry-run release completed$(NC)"

# Check if goreleaser is installed
goreleaser-check: ## Check if goreleaser is installed
	@echo -e "$(BLUE)Checking goreleaser installation...$(NC)"
	@which $(GORELEASER) > /dev/null || (echo -e "$(RED)goreleaser not found. Run 'make tools-install' to install it.$(NC)" && exit 1)
	@echo -e "$(GREEN)✅ goreleaser is installed$(NC)"

# CI/CD targets
ci-build: ## CI build target
	@echo -e "$(BLUE)Running CI build...$(NC)"
	@make deps
	@make fmt
	@make lint
	@make test
	@make build
	@echo -e "$(GREEN)✅ CI build completed$(NC)"

# Development workflow targets
dev-setup: ## Setup development environment
	@echo -e "$(BLUE)Setting up development environment...$(NC)"
	@make tools-install
	@make deps
	@echo -e "$(GREEN)✅ Development environment setup completed$(NC)"

# Status and info targets
status: ## Show project status
	@echo -e "$(BLUE)Project Status:$(NC)"
	@echo "=================="
	@echo -n "Go version: "
	@$(GOCMD) version
	@echo -n "golangci-lint: "
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		echo -e "$(GREEN)✅ available$(NC)"; \
	else \
		echo -e "$(RED)❌ not available$(NC)"; \
	fi
	@echo -n "goreleaser: "
	@if command -v $(GORELEASER) > /dev/null; then \
		echo -e "$(GREEN)✅ available$(NC)"; \
	else \
		echo -e "$(RED)❌ not available$(NC)"; \
	fi
	@echo -n "Docker: "
	@if command -v docker > /dev/null && docker info > /dev/null 2>&1; then \
		echo -e "$(GREEN)✅ available$(NC)"; \
	else \
		echo -e "$(YELLOW)⚠️  not available (local mode only)$(NC)"; \
	fi

# Pipeline status
pipeline-status: ## Show pipeline status and available steps
	@echo -e "$(BLUE)Pipeline Status:$(NC)"
	@echo "=================="
	@if [ -f $(BINARY_NAME) ]; then \
		echo -e "$(GREEN)✅ Binary available$(NC)"; \
		./$(BINARY_NAME) --list-pipelines; \
		echo ""; \
		./$(BINARY_NAME) --list-steps; \
	else \
		echo -e "$(YELLOW)⚠️  Binary not built. Run 'make build' first$(NC)"; \
	fi
