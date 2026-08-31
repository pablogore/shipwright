# Shipwright Makefile
# Usage: make [target] [options]

# GNU Make defaults every recipe to /bin/sh, ignoring the invoking user's
# shell. On macOS /bin/sh is bash in POSIX mode (accepts `set -o pipefail`),
# but on Linux CI runners /bin/sh is dash, which rejects it outright
# ("Illegal option -o pipefail") — caught when the `security` target failed
# on the first real push to develop that exercised it via ci-final, despite
# passing every local run on macOS. Force bash for every recipe so bash-only
# syntax (pipefail, [[ ]], etc.) behaves identically on every platform.
SHELL := /bin/bash

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=shipwright

# Tools
GORELEASER=goreleaser

# Coverage threshold
COVERAGE_THRESHOLD=90
COVERAGE_THRESHOLD_CI=70
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

.PHONY: all build clean test dagger-test test-integration deps tools-install release release-snapshot release-dry-run help coverage coverage-html coverage-report coverage-package coverage-file coverage-summary coverage-threshold coverage-100 local-run pipeline-local build-release build-all-platforms lint ci-final provider-go-standalone

# Help target
.PHONY: help
help: ## Show this help message
	@echo -e "$(BLUE)🚀 Shipwright - Makefile$(NC)"
	@echo "=================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make all                # Complete pipeline: test, build"
	@echo "  make build              # Build the application"
	@echo "  make test               # Run all tests"
	@echo "  make coverage           # Generate complete coverage report"
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
	@echo "  make lint               # Run golangci-lint"
	@echo "  make quality            # Run all quality checks"
	@echo "  make ci-final           # Full production-critical validation contract (build, test, coverage, security)"
	@echo "  make clean              # Clean build artifacts"
	@echo ""
	@echo "Pipeline:"
	@echo "  make pipeline-local     # Run complete pipeline locally"
	@echo "  make pipeline-setup     # Run setup step locally"
	@echo "  make pipeline-build     # Run build step locally"
	@echo "  make pipeline-test      # Run test step locally"

all: test build

build: ## Build the application (root binary + full compile-check of root, providers/go, providers/rust)
	@echo -e "$(BLUE)Building application...$(NC)"
	$(GOBUILD) -o $(BINARY_NAME) .
	@# `go build -o` above only builds the main package; `./...` below is a
	@# side-effect-free compile-check of every root package (go discards
	@# build output when the pattern matches multiple packages, so this is
	@# safe to run from a plain checkout — no stray binaries), matching what
	@# CI's build job already does. providers/go and providers/rust are
	@# separate modules in go.work; `./...` never crosses that boundary, so
	@# each needs its own invocation, same as test/vet/fmt/security.
	$(GOBUILD) ./...
	cd providers/go && $(GOBUILD) ./...
	cd providers/rust && $(GOBUILD) ./...
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
	@# providers/go and providers/rust are separate modules in go.work; `./...`
	@# never crosses that boundary (only the `all` pattern or fully-qualified
	@# paths do, and `all` pulls in the whole dependency graph), so each needs
	@# its own invocation.
	cd providers/go && $(GOTEST) -v -race ./...
	cd providers/rust && $(GOTEST) -v -race ./...
	@echo -e "$(GREEN)✅ Tests completed$(NC)"

dagger-test: ## Run .dagger/'s own tests (separate Go module; deliberately NOT part of `test`/`check`/`quality`/`all` — see design.md D-B isolation)
	@echo -e "$(BLUE)Running .dagger module tests...$(NC)"
	# GOWORK=off: .dagger is not a member of the root go.work, so workspace
	# auto-detection walking up from here fails with "directory prefix .
	# does not contain modules listed in go.work" unless workspace mode is
	# disabled for this invocation.
	cd .dagger && GOWORK=off dagger run $(GOTEST) -race ./...
	@echo -e "$(GREEN)✅ .dagger module tests completed$(NC)"

security-dagger: ## Run govulncheck against .dagger's own Go module (requires a running Dagger engine; deliberately NOT part of `security`/`ci-final` -- see dagger-test's isolation note). .dagger imports the Dagger-generated internal/dagger client, which only exists after `dagger` regenerates it, so plain govulncheck can't resolve the module without a live engine (same reason Dependabot can't update .dagger/go.mod on its own).
	@echo -e "$(BLUE)Running .dagger module security scan...$(NC)"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	cd .dagger && GOWORK=off dagger run govulncheck ./...
	@echo -e "$(GREEN)✅ .dagger module security scan completed$(NC)"

test-integration: ## Run real-Dagger-engine tests under testing/integration/ (requires a running Dagger engine; deliberately NOT part of `test`/`check`/`ci-final` — mock tests cover this logic for the default suite)
	@echo -e "$(BLUE)Running Dagger integration tests...$(NC)"
	$(GOTEST) -tags integration -v -race ./testing/...
	@echo -e "$(GREEN)✅ Integration tests completed$(NC)"

final-sha-gate-check-test: ## Run the final-sha-gate composition logic's own test suite (pure bash, no Dagger required -- included in ci-final)
	@echo -e "$(BLUE)Running final-sha-gate composition logic tests...$(NC)"
	./scripts/final-sha-gate-check_test.sh
	@echo -e "$(GREEN)✅ final-sha-gate composition logic tests completed$(NC)"

deps: ## Download and tidy dependencies
	@echo -e "$(BLUE)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo -e "$(GREEN)✅ Dependencies updated$(NC)"

# Install development tools
tools-install: ## Install development tools (goreleaser)
	@echo -e "$(BLUE)Installing development tools...$(NC)"
	@echo -e "$(YELLOW)Installing goreleaser...$(NC)"
	@$(GOGET) github.com/goreleaser/goreleaser@latest
	@echo -e "$(GREEN)✅ Development tools installed$(NC)"

# Run all checks (test only)
check: test ## Run all checks (test)

# Format code
# providers/go and providers/rust are separate go.work modules; `go fmt all`
# was tried and explicitly refuses it ("not formatting packages in
# dependency modules"), so each needs its own invocation, same as vet/test
# below.
fmt: ## Format code with gofmt
	@echo -e "$(BLUE)Formatting code...$(NC)"
	$(GOCMD) fmt ./...
	cd providers/go && $(GOCMD) fmt ./...
	cd providers/rust && $(GOCMD) fmt ./...
	@echo -e "$(GREEN)✅ Code formatting completed$(NC)"

# Run go vet
# `go vet all` was tried and fails (exit 1) because `all` also pulls in and
# vets third-party dependency *_test.go files with pre-existing, unrelated
# vet issues (e.g. google/go-cmp, prometheus/client_golang) — not a safe
# drop-in. Iterate all modules explicitly instead.
vet: ## Run go vet
	@echo -e "$(BLUE)Running go vet...$(NC)"
	$(GOCMD) vet ./...
	cd providers/go && $(GOCMD) vet ./...
	cd providers/rust && $(GOCMD) vet ./...
	@echo -e "$(GREEN)✅ Go vet completed$(NC)"

# Lint check
# golangci-lint does not span go.work module boundaries either: run from
# providers/go or providers/rust it auto-discovers and applies the root
# .golangci.yml by walking up parent directories (verified via
# `golangci-lint run -v`, which logs "Used config file ../../.golangci.yml"),
# so no explicit --config flag is needed. Iterate all three modules
# explicitly, same pattern as vet/test/security.
lint: ## Run golangci-lint
	@echo -e "$(BLUE)Running golangci-lint...$(NC)"
	@which golangci-lint > /dev/null || (echo -e "$(RED)golangci-lint not found. Install it first (e.g. 'brew install golangci-lint' or see https://golangci-lint.run/welcome/install/).$(NC)" && exit 1)
	golangci-lint run ./...
	cd providers/go && golangci-lint run ./...
	cd providers/rust && golangci-lint run ./...
	@echo -e "$(GREEN)✅ Lint completed$(NC)"

# Security check
# govulncheck does not span go.work module boundaries (verified: findings
# from root, providers/go, and providers/rust are disjoint), so it must run
# once per module.
#
# PROD-001: the previous version of this target concatenated all three
# modules' reports into one vuln_report.txt and passed if the string
# "No vulnerabilities found." appeared ANYWHERE in it. That is a false-pass:
# if providers/go and providers/rust are clean but root is affected, root's
# report never contains that string, but the concatenated file still does
# (from the clean modules), so the old check reported PASS. Verified with a
# real finding (GO-2026-5158 in go.opentelemetry.io/otel, reachable from
# internal/app/health.go). Fixed by (a) trusting govulncheck's own exit code
# per module via pipefail instead of losing it through `| tee`, and (b)
# failing if "Your code is affected" appears anywhere in the combined report,
# instead of requiring the clean string to appear anywhere.
security: ## Run security vulnerability check
	@echo -e "$(BLUE)Running security vulnerability check...$(NC)"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@set -o pipefail; \
	FAILED=0; \
	govulncheck ./... | tee vuln_report.txt || FAILED=1; \
	(cd providers/go && govulncheck ./...) | tee -a vuln_report.txt || FAILED=1; \
	(cd providers/rust && govulncheck ./...) | tee -a vuln_report.txt || FAILED=1; \
	if grep -q "Your code is affected" vuln_report.txt; then \
		FAILED=1; \
	fi; \
	if [ "$$FAILED" -eq 1 ]; then \
		echo -e "$(RED)❌ Security vulnerabilities detected (or scan failed)! Please review.$(NC)"; \
		cat vuln_report.txt; \
		rm -f vuln_report.txt; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ No security vulnerabilities found.$(NC)"; \
		rm -f vuln_report.txt; \
	fi

# Run all code quality checks
quality: fmt vet test security ## Run all code quality checks (fmt, vet, test, security)

# P0 production-readiness gate: providers/go must be independently
# consumable/buildable outside this workspace/monorepo. `build`/`test` above
# also `cd providers/go`, but go's workspace-mode auto-detection walks up
# parent directories looking for go.work, so those targets still silently
# benefit from go.work resolving github.com/pablogore/shipwright against the
# local checkout -- exactly what let providers/go/go.mod pin a stale
# pseudo-version (predating RuntimeInspector/RuntimeUpgrader/DriftReport)
# without any in-workspace build or test ever catching it. GOWORK=off is the
# only way to prove providers/go/go.mod's own require directives are
# sufficient on their own, the way an external consumer or the module proxy
# would resolve them. A `replace` directive would defeat the same proof by
# a different route, so it's rejected here too.
provider-go-standalone: ## Validate providers/go builds/tests standalone with GOWORK=off (no workspace, no replace)
	@echo -e "$(BLUE)Validating providers/go is standalone (GOWORK=off)...$(NC)"
	@if grep -q '^replace' providers/go/go.mod; then \
		echo -e "$(RED)❌ providers/go/go.mod has a replace directive -- module is not independently consumable$(NC)"; \
		exit 1; \
	fi
	cd providers/go && GOWORK=off $(GOMOD) download
	cd providers/go && GOWORK=off $(GOMOD) verify
	cd providers/go && GOWORK=off $(GOBUILD) ./...
	cd providers/go && GOWORK=off $(GOTEST) -race ./...
	@echo -e "$(GREEN)✅ providers/go is standalone under GOWORK=off$(NC)"

# PROD-001: single repository-owned source of truth for "what makes a SHA
# production-ready", run identically by any CI provider or locally.
# ci-final = lint + build + test + coverage + security. Three things are
# deliberately excluded from that composition (or from the coverage
# calculation); each is documented below as: why it's out, debt vs.
# architectural decision, the open risk, and what closes it.
#
# `lint` was excluded here as pre-existing debt (110 findings across root,
# providers/go, providers/rust) tracked by
# https://github.com/pablogore/shipwright/issues/186. That issue is now
# resolved: `make lint` passes with 0 findings in all three modules (the 5
# providers/rust exported-type "stutter" names are intentional — see their
# `//nolint:revive` rationale at each declaration — and are not renamed,
# since that would be a breaking public API change to a separately,
# tag-released module). `lint` is included in ci-final's composition below.
#
# 1) `dagger-test` — EXCLUDED, ARCHITECTURAL DECISION (permanent).
#    `.dagger/`'s own module tests require `dagger run`, i.e. a live Dagger
#    engine — see design.md D-B isolation. A fail-closed, CI-provider-
#    independent gate cannot depend on an engine being reachable at
#    validation time. Risk left open: none beyond what `test-integration`
#    (below) already covers for real-engine behavior; `.dagger/`'s own logic
#    is still exercised by `make dagger-test` on demand, just not gated.
#    Closes via: N/A — this exclusion cannot be lifted without dropping the
#    "runs identically anywhere, no live engine required" property ci-final
#    exists to guarantee.
#
# 2) `test-integration` — EXCLUDED, ARCHITECTURAL DECISION (permanent).
#    `testing/integration/{go,rust,changelog}/` (build tag `integration`)
#    exercises the real `dagger.io/dagger` SDK against a live engine, by
#    design — that's the point of separating it from the daggerkit-mocked
#    unit suite. Same fail-closed/reproducibility argument as `dagger-test`.
#    Risk left open: daggerkit's interface/adapter layer could in principle
#    drift from real SDK behavior without ci-final noticing; that drift is
#    caught by `make test-integration` run on demand (and in CI where an
#    engine is available), not by the gate itself.
#    Closes via: N/A — same reasoning as `dagger-test`.
#
# 3) `internal/daggerkit` coverage exclusion — ARCHITECTURAL DECISION.
#    Excluded from the coverage calculation (both `coverage` and
#    `coverage-gate`, see below) alongside `/mocks`, `/examples`, `/app`,
#    `/config`. daggerkit's adapter code is a thin, mechanical passthrough to
#    the real Dagger SDK types (see internal/daggerkit/adapter_test.go for
#    what IS tested: pure roundtrips and type-assertion guards); the
#    passthrough branches themselves are only meaningfully exercised against
#    a live engine, i.e. by `test-integration`, not by a coverage-counted
#    unit test. Risk left open: none — this mirrors the existing,
#    already-accepted treatment of `/mocks` and `/app` for the same reason
#    (thin wrappers around real/external dependencies).
#    Closes via: N/A — permanent, same category as the existing /mocks and
#    /app exclusions.
#
# 4) ci-final gates on `coverage-gate`, NOT `coverage` — CORRECTED DEFECT,
#    found by the first real push to develop after this gate went live.
#    `coverage` enforces COVERAGE_THRESHOLD=90, a bar this repo's own total
#    coverage has never reached (see coverage-ci, added in e85a137 with
#    COVERAGE_THRESHOLD_CI=70 specifically because 90% total was never
#    realistic to enforce automatically; also documented as the "90% local /
#    70% CI" convention in openspec/changes/archive/2026-08-26-.../tasks.md).
#    ci-final originally depended on `coverage` (90%, fail-closed), which
#    meant the Final-SHA gate could never pass for any SHA — caught when
#    SHA 041361d failed final-sha-validation on the first real push to
#    develop at 73.6% total coverage. `coverage-gate` is the fail-closed
#    counterpart of the pre-existing `coverage-ci` (same 70% bar, but exits 1
#    instead of only warning); `coverage` (90%) stays as-is for local/
#    aspirational use, `coverage-ci` stays as-is for its existing callers.
#    Closes via: already closed by this correction.
#
# 5) `provider-go-standalone` -- INCLUDED, P0 REGRESSION GATE.
#    `build`/`test` above compile-check and run providers/go's own tests, but
#    both do so with go.work in effect (workspace-mode auto-detection walks
#    up from providers/go and finds it), which resolves github.com/pablogore/
#    shipwright against the local checkout regardless of what providers/go/
#    go.mod actually requires. That's exactly how a stale pseudo-version
#    (predating RuntimeInspector/RuntimeUpgrader/DriftReport) shipped
#    undetected until the release-provider-go.yml tag workflow broke on a
#    real GOWORK=off build. Included here so every push to develop -- not
#    just a release tag push -- catches this class of drift immediately.
#    Closes via: already closed by this gate.
#
# Given the above, ci-final represents the valid minimum production-critical
# set: lint/build/test/coverage/security/provider-go-standalone are the only
# checks that (a) can run fail-closed with zero external dependencies and
# (b) map directly to "this SHA compiles cleanly, behaves correctly under
# test, meets its coverage bar, has no known vulnerabilities, and every
# independently-published module it contains is actually independent."
# dagger-test/test-integration structurally cannot be, by the same
# reproducibility requirement that motivates ci-final's existence.
.PHONY: ci-final
ci-final: lint build test coverage-gate security provider-go-standalone final-sha-gate-check-test ## Full production-critical validation contract for the Final SHA (repository-owned, CI-provider independent; see comment above)
	@echo -e "$(GREEN)✅ ci-final: all production-critical guards passed$(NC)"

# Coverage targets
# NOTE: all `go list ./...`-based package lists below intentionally do NOT
# include providers/go. That is a deliberate scope decision, not the ./...
# workspace-traversal gap fixed elsewhere in this file (test/vet/fmt/security):
# providers/go is a separately versioned, externally-published module (see
# openspec/changes/shipwright-provider-go-module/design.md D2/D4) whose own
# coverage story is not calibrated against root's COVERAGE_THRESHOLD values
# and does not belong merged into root's coverage.out. It gets its own tests
# run (see `test` above); a dedicated coverage target for it, analogous to
# `dagger-test`'s isolation, is future work if/when it needs one.
#
# The filters also exclude /daggerkit: each module's daggerkit package is a
# thin adapter over the real dagger.io/dagger SDK types. Its mocks are
# exercised by consumer packages' tests (cross-package usage that Go's
# default per-package coverage doesn't credit without -coverpkg), and its
# adapter methods mostly delegate straight into the real SDK, so they can
# only be proven against a live engine — that's exactly what
# testing/integration/ (make test-integration) is for, not this local gate.
coverage: ## Generate comprehensive ASCII coverage report with threshold validation
	@echo -e "$(BLUE)Generating comprehensive coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                           📊 COVERAGE SUMMARY                                ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
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
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo -e "$(GREEN)✅ HTML coverage report generated: coverage/coverage.html$(NC)"

coverage-report: coverage-package coverage-file ## Generate detailed coverage reports by package and file

coverage-package: ## Generate detailed ASCII coverage report by package
	@echo -e "$(BLUE)Generating package coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        📦 PACKAGE COVERAGE REPORT                            ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)Package Coverage Breakdown:$(NC)"
	@echo -e "$(WHITE)────────────────────────────$(NC)"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep -v "/app/" | grep -v "/config/" | \
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
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total Package Coverage: $${COVERAGE}%$(NC)"

coverage-file: ## Generate detailed ASCII coverage report by file
	@echo -e "$(BLUE)Generating file coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                         📄 FILE COVERAGE REPORT                              ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)File Coverage Breakdown:$(NC)"
	@echo -e "$(WHITE)─────────────────────────$(NC)"
	@$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "\.go:" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | \
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
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total File Coverage: $${COVERAGE}%$(NC)"

coverage-summary: ## Generate comprehensive coverage summary with statistics
	@echo -e "$(BLUE)Generating comprehensive coverage summary...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        📈 COMPREHENSIVE COVERAGE SUMMARY                     ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(WHITE)Coverage Statistics:$(NC)"
	@echo -e "$(WHITE)────────────────────$(NC)"
	@TOTAL_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	PACKAGES=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | wc -l); \
	FILES=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "\.go:" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | wc -l); \
	HIGH_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | awk '{gsub(/%/, "", $$3); if ($$3 >= 90) print $$1}' | wc -l); \
	MEDIUM_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | awk '{gsub(/%/, "", $$3); if ($$3 >= 80 && $$3 < 90) print $$1}' | wc -l); \
	LOW_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | awk '{gsub(/%/, "", $$3); if ($$3 < 80) print $$1}' | wc -l); \
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
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        🎯 COVERAGE THRESHOLD VALIDATION                      ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@TOTAL_COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(WHITE)Threshold Validation:$(NC)"; \
	echo -e "$(WHITE)─────────────────────$(NC)"; \
	echo -e "$(BLUE)Required Threshold: $(COVERAGE_THRESHOLD)%$(NC)"; \
	echo -e "$(BLUE)Current Coverage: $${TOTAL_COVERAGE}%$(NC)"; \
	echo ""; \
	if [ $$(echo "$$TOTAL_COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo -e "$(RED)❌ FAILED: Coverage $${TOTAL_COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		echo ""; \
		echo -e "$(YELLOW)Packages below threshold:$(NC)"; \
		$(GOCMD) tool cover -func=coverage/coverage.out | grep -E "gitlab.com/syntegrity" | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | \
		awk -v threshold=$(COVERAGE_THRESHOLD) '{ \
			coverage = $$3; \
			gsub(/%/, "", coverage); \
			if (coverage < threshold) printf "$(RED)%-60s %6s$(NC)\n", $$1, $$3; \
		}'; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ PASSED: Coverage meets threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
	fi

coverage-ci: ## Generate coverage report for CI with relaxed threshold
	@echo -e "$(BLUE)Generating CI coverage report...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@echo ""
	@echo -e "$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(CYAN)║                        📊 CI COVERAGE SUMMARY                                 ║$(NC)"
	@echo -e "$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	echo -e "$(GREEN)✅ Total Coverage: $${COVERAGE}%$(NC)"; \
	if [ -n "$$COVERAGE" ] && [ "$$COVERAGE" != "" ]; then \
		if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD_CI)" | bc -l 2>/dev/null || echo "1") -eq 1 ]; then \
			echo -e "$(YELLOW)⚠️  Coverage $${COVERAGE}% is below CI threshold $(COVERAGE_THRESHOLD_CI)%$(NC)"; \
			echo -e "$(YELLOW)   This is acceptable for CI but should be improved over time$(NC)"; \
		else \
			echo -e "$(GREEN)✅ Coverage meets CI threshold $(COVERAGE_THRESHOLD_CI)%$(NC)"; \
		fi; \
	else \
		echo -e "$(YELLOW)⚠️  Could not determine coverage percentage$(NC)"; \
	fi

# coverage-gate is the fail-closed counterpart of coverage-ci: same 70% CI
# threshold (COVERAGE_THRESHOLD_CI), but exits 1 instead of only warning.
# `coverage` (90%) is the local/aspirational bar — never realistic as an
# automated gate, which is exactly why coverage-ci exists — so ci-final
# must not depend on it directly. This is the target ci-final actually uses.
coverage-gate: ## Fail-closed coverage validation at the CI threshold (used by ci-final)
	@echo -e "$(BLUE)Validating coverage against the CI gate threshold...$(NC)"
	@mkdir -p coverage
	@$(GOTEST) -coverprofile=coverage/coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples | grep -v /mocks | grep -v /app | grep -v /config | grep -v /daggerkit)
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage/coverage.out | grep -v "/mocks/" | grep -v "/examples/" | grep -v "/proto/" | grep -v "/app/" | grep -v "/config/" | grep -v "/daggerkit/" | grep total | awk '{print $$3}' | sed 's/%//' | sed 's/(statements)//' | tr -d ' '); \
	if [ -z "$$COVERAGE" ]; then \
		echo -e "$(RED)❌ Could not determine coverage percentage$(NC)"; \
		exit 1; \
	fi; \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD_CI)" | bc -l 2>/dev/null || echo "1") -eq 1 ]; then \
		echo -e "$(RED)❌ Coverage $${COVERAGE}% is below CI gate threshold $(COVERAGE_THRESHOLD_CI)%$(NC)"; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ Coverage $${COVERAGE}% meets CI gate threshold $(COVERAGE_THRESHOLD_CI)%$(NC)"; \
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

# Release build targets
build-release: ## Build release binary with version info
	@echo -e "$(BLUE)🔨 Building release binary...$(NC)"
	@mkdir -p release
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	$(GOBUILD) -ldflags="-X main.Version=$$VERSION -X main.BuildTime=$$BUILD_TIME -X main.GitCommit=$$GIT_COMMIT" -o release/$(BINARY_NAME) .
	@echo -e "$(GREEN)✅ Release binary built: release/$(BINARY_NAME)$(NC)"
	@echo -e "$(CYAN)Version: $$(./release/$(BINARY_NAME) --version)$(NC)"

build-all-platforms: ## Build binaries for all supported platforms
	@echo -e "$(BLUE)🔨 Building binaries for all platforms...$(NC)"
	@mkdir -p release
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	LDFLAGS="-X main.Version=$$VERSION -X main.BuildTime=$$BUILD_TIME -X main.GitCommit=$$GIT_COMMIT"; \
	echo "Building for Linux AMD64..."; \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$$LDFLAGS" -o release/$(BINARY_NAME)-linux-amd64 .; \
	echo "Building for Linux ARM64..."; \
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$$LDFLAGS" -o release/$(BINARY_NAME)-linux-arm64 .; \
	echo "Building for macOS AMD64..."; \
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$$LDFLAGS" -o release/$(BINARY_NAME)-darwin-amd64 .; \
	echo "Building for macOS ARM64..."; \
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$$LDFLAGS" -o release/$(BINARY_NAME)-darwin-arm64 .; \
	echo "Building for Windows AMD64..."; \
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$$LDFLAGS" -o release/$(BINARY_NAME)-windows-amd64.exe .; \
	echo -e "$(GREEN)✅ All platform binaries built in release/ directory$(NC)"
	@echo -e "$(CYAN)Generated files:$(NC)"
	@ls -la release/
