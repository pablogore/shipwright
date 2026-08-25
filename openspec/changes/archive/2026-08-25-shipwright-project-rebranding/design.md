# Design: Shipwright Project Rebranding

> SPEC-000 · Mechanical identity rename. No architecture is introduced or changed.

## Technical Approach

No new architecture. The work is an ordered, token-scoped textual substitution over
~905 occurrences in ~105 files, sequenced by *dependency*: the Go module path first
(nothing else can compile or be verified until imports resolve), then each downstream
identity surface. Every phase ends compiling; the last phase ends with a residual sweep.

Two mechanisms carry the whole change: an **ordered longest-match-first token map**
(prevents partial rewrites) and an explicit **deny-list of preserved identities**
(prevents collateral damage to the real `getsyntegrity` company and the Dagger SDK).

## Identity Resolution Chain (what depends on what)

    go.mod module path ──→ 56 .go imports ──→ compiles ──→ everything below verifiable
            │                                                  │
            ├──→ Makefile BINARY_NAME ──→ .goreleaser.yml ──→ release.yml artifact names
            ├──→ main.go flagset/help/version ──→ docs + examples CLI snippets
            ├──→ config.EnvPrefix ──→ SHIPWRIGHT_* ──→ dependabot secret + CI env
            └──→ yaml_parser candidate list ──→ .shipwright.yml ──→ docs/examples configs

## Ordered Token Map (apply strictly top-to-bottom)

| # | Match (literal) | Replace |
|---|---|---|
| 1 | `github.com/getsyntegrity/syntegrity-dagger` | `github.com/pablogore/shipwright` |
| 2 | `github.com/pablogore/syntegrity-dagger` | `github.com/pablogore/shipwright` |
| 3 | `getsyntegrity/syntegrity-dagger` | `pablogore/shipwright` |
| 4 | `.syntegrity-dagger.` / `syntegrity-dagger.y` (config files) | `.shipwright.` / `shipwright.y` |
| 5 | `SYNTEGRITY_DAGGER_` | `SHIPWRIGHT_` |
| 6 | `SYNTEGRITY_DAGGER` | `SHIPWRIGHT` |
| 7 | `Syntegrity Dagger` | `Shipwright` |
| 8 | `SyntegrityDagger` | `Shipwright` |
| 9 | `syntegrity_dagger` | `shipwright` |
| 10 | `syntegrity-dagger` (catch-all, last) | `shipwright` |

Order is load-bearing: rule 10 applied first would leave `github.com/getsyntegrity/shipwright`.
Every rule requires the `syntegrity`+`dagger` token pair, so bare `dagger` is never a target.

Manual (pattern does not match): `internal/config/errors.go:1` and
`internal/config/appconf.test.go:1` — "aplicación Syntegrity" → "aplicación Shipwright".

## Preserved Identities (deny-list — must show zero diff)

| Preserved | Why safe |
|---|---|
| `dagger.io/dagger` (~19 files) | No `syntegrity` token; unreachable by all 10 rules |
| `github.com/getsyntegrity/go-kit-logger`, `go.sum` | Rule 1 needs the `/syntegrity-dagger` suffix |
| `Syntegrity CI`, `getsyntegrity.com`, `$HOME/.ssh/syntegrity` (`shared/*_cloner.go`) | Company identity |
| `SyntegrityInfraPipeline`, `syntegrity-infra` (`internal/pipelines/infra/`) | Company infra domain name, no `dagger` token |
| `ghcr.io/syntegrity` (`examples/configs/tenant-svc.yml`), `AGENTS.md` eventengine | Company-owned examples |
| Makefile `gitlab.com/syntegrity` grep, root `1export` | Documented pre-existing follow-ups |
| `openspec/changes/shipwright-project-rebranding/**`, `.git/`, `coverage/` | SDD artifacts quote the old name by design |

## Execution Phases

| P | Scope | Gate |
|---|---|---|
| 0 | Freeze exclusion list; capture baseline grep counts and coverage % | Baseline recorded |
| 1 | `go mod edit -module`; imports in `.go`, `mocks/`, `tests/`, `examples/` | `go build ./...` |
| 2 | Makefile `BINARY_NAME`, `.goreleaser.yml`, `main.go` flagset/help/version | `make build` |
| 3 | `config.EnvPrefix`, `yaml_parser.go` 6-entry candidate list, `main.go -config` default | `go test ./internal/config/...` |
| 4 | `git mv .github/actions/syntegrity-dagger shipwright`; `ci.yml`, `release.yml`, `dependabot.yml`, `CODEOWNERS`, `rulesets/README.md`, `scripts/`, `.gitignore` | yamllint |
| 5 | `README.md` + badge, 21 `docs/`, `CHANGELOG.md`, `examples/**` | Link check |
| 6 | Comments, `.serena/project.yml`, `openspec/config.yaml` build/test commands | gofmt |
| 7 | Residual sweep + full verification | See below |

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Module path | `github.com/pablogore/shipwright` | Keep `getsyntegrity/*` org | `go.mod` org is already stale vs. the real remote; product ≠ company |
| Replacement mechanism | Ordered literal token map, longest-first | Single broad `s/syntegrity/shipwright/gi` | A broad rule destroys `go-kit-logger`, `SyntegrityInfraPipeline`, SSH/registry defaults |
| Config filename | Rename all 6 discovery candidates (`.yml`/`.yaml`, bare, `.github/`) | Rename only `.shipwright.yml` | Partial rename would silently keep old-name discovery paths alive |
| Compatibility | Clean break, no aliases or dual env-prefix fallback | Read both prefixes for one release | Proposal decided clean rename; a fallback would be new behavior, violating atomicity |
| Phase 1 first | Module/imports before all else | Docs-first | Compiler is the only automatic verifier of the largest slice (56 files) |
| TDD ordering | RED-first on the three behavior-bearing surfaces (env prefix, config discovery, binary/help) | Rename production code then fix tests | `strict_tdd: true`; those surfaces are observable contracts, docs are not testable |

## Testing Strategy

| Layer | What | How |
|---|---|---|
| Unit (RED-first) | `EnvPrefix` = `SHIPWRIGHT_`; `findConfigFile` candidate list | Update `internal/config/config_test.go`, `yaml_parser_test.go` to assert new values → fail → rename |
| Unit | Existing suites unchanged in intent | `go test -race ./...`, coverage ≥ 90 (unchanged threshold) |
| Compile | Imports across `internal/`, `mocks/`, `tests/`, `examples/` | `go build ./...` (not `go build .` — `examples/` is `package main` and only `./...` compiles it) |
| Integration | CLI surface | `./shipwright --help`, `--version` contain no old token |
| Non-regression | Deny-list untouched | `git diff` over `dagger.io/dagger` and `go-kit-logger` import lines must be empty |
| Sweep | Residual | `rg -i 'syntegrity[-_ ]?dagger'` returns only deny-list paths; `rg -i syntegrity` returns only company-identity hits |

## Threat Matrix

`N/A` — no routing, subprocess, VCS/PR automation, executable-file classification, or
process-integration boundary changes. Shell scripts, CI workflows, and the binary name
change *identifiers only*; control flow and invocation semantics are byte-equivalent.
Rename tooling itself must not run in-place over `.git/`, `go.sum`, `coverage/`, or binaries.

## Migration / Rollout

No data or state migration. Two out-of-band operational prerequisites, both non-code:

1. Rename the GitHub repository `pablogore/syntegrity-dagger` → `pablogore/shipwright`
   (and update the git remote) so `go get` and release URLs resolve.
2. Create repository secret `SHIPWRIGHT_TOKEN` before merge — `.github/dependabot.yml`
   references `secrets.SYNTEGRITY_DAGGER_TOKEN`; renaming the reference without creating
   the secret breaks Dependabot's private-registry auth.

Rollback: single-purpose branch, no functional edits — `git revert -m 1 <merge-sha>`
restores the old identity wholesale. If a release already shipped under the new name,
cut a corrected tag.

## Open Questions

- [ ] Timing of the GitHub repo rename relative to merge (before vs. same-day) — owner decision, not a code blocker.
- [ ] Whether the old repo name should keep GitHub's automatic redirect enabled, or be re-claimed. Non-blocking.
