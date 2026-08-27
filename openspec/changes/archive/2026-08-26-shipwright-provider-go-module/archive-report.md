# Archive Report — shipwright-provider-go-module

**Change**: `shipwright-provider-go-module`
**Archived**: 2026-08-26
**Status**: COMPLETE, PASS (0 CRITICAL, 0 WARNING)

---

## Executive Summary

The Go provider extraction is complete. `providers/go` is now a standalone, independently-versioned Go module (`github.com/pablogore/shipwright/providers/go`) published at v0.1.0, consumed by the root Shipwright module without any local `replace` directive.

The change was delivered across 3 PR slices:
1. **Slice 1** (PR #164): Extraction — moved 5 source files + 8 tests from `internal/capabilities/` to `providers/go/`, swapped importers, added workspace guards
2. **Slice 1b** (PR #165): Release automation — tag-scoped CI workflow, goreleaser isolation, tag-namespace guard
3. **Slice 2** (PR #173): Removed `replace` directive, added no-replace guard test, verified proxy resolution

Additionally, PR #172 corrected the OpenSpec tag target to point to the #165 merge commit (documentary fix, not a design slice).

---

## Scope and Deliverables

### What Was Delivered

| # | Deliverable | Status |
|---|---|---|
| 1 | `providers/go/` standalone module with own `go.mod` | Complete |
| 2 | 5 capability implementations extracted (GoBuilder, GoLinter, GoUnitTester, GoVulnScanner, ContainerPublisher) | Complete |
| 3 | `go.work` workspace with allowlisted members (`.`, `./providers/go`) | Complete |
| 4 | Root `go.mod` imports `providers/go@v0.1.0` from proxy (no replace) | Complete |
| 5 | Workspace guard tests (use-set, dagger-isolation, replace-directive, CI/Makefile) | Complete |
| 6 | Release automation: `release-provider-go.yml` with shape/identity/isolation/proxy gates | Complete |
| 7 | Tag `providers/go/v0.1.0` published and resolvable from `proxy.golang.org` | Complete |

### Design Decisions (from design.md)

- **D1**: `go.work` isolates modules; `./...` never crosses boundaries
- **D2**: `providers/go` is a separately-versioned module, not a nested package
- **D4**: Temporary `replace` directive sanctioned for slice 1 only; slice 2 removes it
- **D5**: No `internal/**` imports from `providers/go` (enforced by `internalimport_test.go`)
- **D6**: Tag-scoped release workflow for provider modules
- **D-F**: Flat package `golang` (not `go`, a keyword), no bundle identity

---

## Verification Evidence

All checks from verify-report.md confirmed:
- Zero replace directives in root `go.mod`
- `providers/go@v0.1.0` resolves from module proxy
- Build, vet, test clean on both modules (short mode; Dagger integration tests require Docker daemon)
- Diamond workflow regression passes
- Guard test `TestRootGoModHasNoReplaceDirectives` GREEN

---

## Artifact Observations

| Artifact | Location |
|---|---|
| Proposal | `openspec/changes/shipwright-provider-go-module/proposal.md` |
| Design | `openspec/changes/shipwright-provider-go-module/design.md` |
| Tasks | `openspec/changes/shipwright-provider-go-module/tasks.md` (all 9 phases, 30+ tasks checked) |
| Verify Report | `openspec/changes/shipwright-provider-go-module/verify-report.md` |
| Specs | `openspec/specs/public-module-api/spec.md` (updated for providers/go) |

---

## Archive Metadata

- **Archived by**: sdd-archive executor
- **Archive date**: 2026-08-26
- **Archive path**: `openspec/changes/archive/2026-08-26-shipwright-provider-go-module/`
- **Specs touched**: `openspec/specs/public-module-api/`
- **PRs**: #164 (slice 1 extraction), #165 (slice 1b release automation), #172 (tag target correction), #173 (slice 2 finalization)
- **Tag**: `providers/go/v0.1.0` @ `ba011826da6b1a672e06396a0335f8fee27ed041`
