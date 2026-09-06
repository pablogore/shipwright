# Apply Progress: Shipwright Project Rebranding

## Work Unit 1 (PR1) — Module Path & Imports (Phase 0 + Phase 1)

Status: **done**
Branch: `feature/shipwright-rebrand-pr1-module-imports` (local, not pushed)

### Phase 0 — Baseline & Exclusion Freeze

- 0.1 Baseline counts (excluding `openspec/changes/shipwright-project-rebranding/**`,
  `.git/`, `coverage/`):
  - `rg -ic 'syntegrity[-_ ]?dagger'`: **669** matching lines across **91** files.
  - `rg -ic syntegrity`: **781** matching lines across **105** files.
  - `make coverage`: **fails at baseline** (before any edit) — the sandbox has no
    network credentials to fetch the private module
    `github.com/getsyntegrity/go-kit-logger@v0.0.0-20250828114729-566d9913c10b`
    (`git ls-remote` over HTTPS requires a GitHub username/password prompt, which
    is disabled in this environment). Packages that don't depend on
    `go-kit-logger` did report coverage: `internal/cache` 57.5%, `internal/interfaces`
    100.0%, `internal/pipelines` 70.8%. This is a pre-existing environment
    limitation, confirmed reproducible identically before and after the rename —
    not a regression introduced by this change.
- 0.2 Deny-list frozen and confirmed untouched (verified post-edit via targeted
  `rg` spot checks): go-kit-logger imports, `SyntegrityInfraPipeline`/`syntegrity-infra`
  in `internal/pipelines/infra/`, `ci@getsyntegrity.com` and `$HOME/.ssh/syntegrity`
  in `internal/pipelines/shared/{ssh,https}_cloner.go`. `AGENTS.md`, the Makefile
  `gitlab.com/syntegrity` grep, root `1export`, `examples/configs/tenant-svc.yml`,
  and the entire `openspec/changes/shipwright-project-rebranding/**` directory were
  not opened for edit at all.

### Phase 1 — Module Path & Imports

- 1.1 `go mod edit -module github.com/pablogore/shipwright` applied to `go.mod`.
  `go.sum` unchanged (no diff — confirmed).
- 1.2 Import paths updated across **54** `.go` files (design's ~56 estimate; actual
  count matching rule 1 `github.com/getsyntegrity/syntegrity-dagger` → `github.com/pablogore/shipwright`)
  under `internal/app`, `internal/config`, `internal/executors`, `internal/pipelines`,
  `internal/plugins`, `mocks/`, `tests/`, `examples/`, plus `main.go`. Applied with
  a literal, scoped `sd` substitution of the exact rule-1 token — no broad regex.
  One additional bare-token occurrence (design rule 3, `getsyntegrity/syntegrity-dagger`
  → `pablogore/shipwright`, no `github.com/` prefix) was found in an SSH-URL test
  fixture at `internal/app/health_test.go:123` and corrected by hand. Confirmed zero
  residual `syntegrity-dagger` hits anywhere else in `.go` files after these two passes.
  `internal/pipelines/shared/*.go` was checked and correctly has **no** module-path
  imports to rewrite (it only imports `go-kit-logger`, which is deny-listed).
- 1.3 [HIGH RISK] `internal/executors/{selector.go,selector_test.go,docker_executor.go,docker_executor_test.go}`
  — only the import path line changed in each file (verified via `git diff`).
- 1.4 [HIGH RISK] `internal/pipelines/**` (`test/`, `infra/`, `go-service/`) — only
  import path lines changed. `infra/pipeline.go` and `infra/pipeline_test.go` keep
  `SyntegrityInfraPipeline` (type name, 15+ occurrences), `"syntegrity-infra"` (Name()
  string + log message + test assertions), and `shared/{ssh,https}_cloner.go` keep
  `ci@getsyntegrity.com` and `$HOME/.ssh/syntegrity` byte-identical — confirmed via
  targeted `rg` diff against the untouched baseline strings.
  One post-edit `gofmt` import reordering was applied to `tests/go_pipeline_test.go`
  (alphabetical import-block reorder caused directly by the new module path sorting
  differently against `github.com/onsi/*`) — this only reordered import lines, no
  other change. Three other files (`internal/config/yaml_parser.go`,
  `internal/pipelines/go-service/pipeline_test.go`, `internal/pipelines/test/gotester.go`)
  showed **pre-existing, unrelated** `gofmt` diffs (struct-tag/field alignment,
  operator spacing) confirmed via `git diff` to be outside the single import-path
  line I touched in each — left untouched per the "only the import path line
  changes" HIGH RISK constraint.
- 1.5 Gate `go build ./...`: **partially verified, blocked by sandbox network
  access**, not by this change. `go build ./...` fails identically before and
  after the rename with the same `go-kit-logger` fetch error (see 0.1). Packages
  reachable without that dependency were built and vetted clean against the new
  module path: `go build ./internal/cache/... ./internal/interfaces/... ./internal/pipelines`
  and `go vet` on the same set both exit 0. `internal/config` also builds clean
  (its test package additionally imports `internal/pipelines/shared`, which pulls
  in `go-kit-logger`, so `go test ./internal/config/...` hits the same network
  wall). No new compile errors were introduced by the rename in any package.
- 1.6 Gate: `git diff -- '*.go' | rg '^[+-].*dagger\.io/dagger'` → empty.
  `git diff -- '*.go' | rg '^[+-].*go-kit-logger'` → empty. Both confirmed clean.

### Diff summary

55 files changed (54 `.go` files + `go.mod`), 92 insertions / 92 deletions
(184 changed lines total) — well under the 400-line chained-PR budget guard.
`go.sum` untouched.

### Known environment limitation (affects this and all later work units)

This sandbox cannot reach `github.com/getsyntegrity/go-kit-logger` over HTTPS
(no git credentials, terminal prompts disabled) and it is not cached in the
local module proxy/download cache. `go build ./...`, `go test -race ./...`,
and `make coverage` will all fail on **any** package that transitively imports
`internal/pipelines/shared` or `go-kit-logger` directly — this includes
`main`, `internal/app`, `internal/executors`, `internal/plugins`, and most of
`internal/pipelines/*`. This reproduces identically on the untouched baseline
(`develop`, pre-rename) and is therefore not a defect introduced by this SDD
change. Later work units (Phase 2 onward) inherit the same constraint for any
gate requiring a full `go build`/`go test`/`make build`/`make coverage`.
This should be flagged to the user/CI owner — CI presumably has proper GitHub
credentials configured and will not hit this limitation.

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 2 | CLI/Build identity (Makefile `BINARY_NAME`, `main.go` flagset/help/version, `.goreleaser.yml`) | PR 2 |
| 3 | Config & env identity (`EnvPrefix`, `yaml_parser.go` 6-entry candidate list, `main.go -config` default) — RED-first | PR 3 |
| 4 | CI/CD & release (`.github/actions/syntegrity-dagger` → `shipwright`, workflows, `dependabot.yml`, `CODEOWNERS`) | PR 4 |
| 5 | Docs & examples (`README.md`, 21 `docs/` files, `CHANGELOG.md`, `examples/**`) | PR 5 |
| 6 | Comments, fixtures, manual edits (`errors.go`/`appconf.test.go` Spanish comments, `.serena/project.yml`, `openspec/config.yaml`) | PR 6 |
| 7 | Residual sweep | PR 7 |
| 8 | Final build + test verification | PR 7 |

Note: the residual `syntegrity-dagger` occurrences confirmed still present after
this work unit (correctly out of scope here) live in `main.go` (flagset name,
`-config` default, help example), `main_test.go` (binary name assertion),
`internal/app/health.go` (User-Agent string), `internal/config/yaml_parser.go`
(6-entry candidate list + doc comment), and `internal/config/yaml_parser_test.go`
(matching test fixtures) — these belong to Phase 2/3, not Phase 1.

## Work Unit 2 (PR2) — CLI / Build (Phase 2)

Status: **done**
Branch: `feature/shipwright-rebrand-pr2-cli-build` (local, not pushed;
stacked on `feature/shipwright-rebrand-pr1-module-imports` per
`stacked-to-main` chain strategy)

### Mode

Strict TDD — RED-first per tasks.md Phase 2 header (`[RED-first: binary/help/version]`).

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1/2.2 | `main_test.go::TestCLIIdentityConstants` | Unit | N/A (new test; pre-existing `main_test.go` tests unaffected in intent) | ✅ Written — asserted `cliName`/`versionLogMessage`/`initLogMessage` before they existed in `main.go`, guaranteeing a compile-time failure | ✅ Confirmed by inspection (execution blocked — see Environment Constraint below) | ✅ 6 assertions: 3 positive equality (happy path per constant) + 3 `NotContains` (negative/edge case proving old identity is gone) | ✅ Extracted 3 magic strings (`cliName`, `versionLogMessage`, `initLogMessage`) into named constants; help-example string reuses `cliName` instead of duplicating the literal |
| 2.1/2.2 | `main_test.go::TestMain_ErrorOutput` (existing) | Unit | N/A — pre-existing smoke test, only its `os.Args[0]` literal changed (`"syntegrity-dagger"` → `cliName`) | ✅ Same RED as above (shared symbol) | ✅ Confirmed by inspection | ➖ Single — literal substitution only, no new branch | ➖ None needed |

### Test Summary

- **Total tests written**: 1 new (`TestCLIIdentityConstants`), 1 existing test updated (`TestMain_ErrorOutput`)
- **Total tests passing**: Not executable in this sandbox (see Environment Constraint). Verified GREEN by structural/compile-time reasoning: after the GREEN edit, `cliName`, `versionLogMessage`, `initLogMessage` are defined in `main.go` with exactly the asserted values (confirmed via `git diff` + `rg` below), so `TestCLIIdentityConstants` would pass if executed.
- **Layers used**: Unit (2)
- **Approval tests**: None — no refactoring-of-existing-behavior tasks in this phase
- **Pure functions created**: 0 (constants only; no new functions — `cliName+" --pipeline go-service --step build"` is a literal concatenation, not extracted into a function, since it has exactly one call site)

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test -run TestCLIIdentityConstants ./.` — **blocked**: fails identically to baseline with `github.com/getsyntegrity/go-kit-logger@...: invalid version: git ls-remote ... fatal: could not read Username for 'https://github.com': terminal prompts disabled` (confirmed via direct `go vet .` and `go build .` runs, both fail at `main.go:11:2` module-download resolution, before reaching any test/type-check of my changes). This is the exact same error reproduced on `develop` (unmodified) per PR1's baseline record — not a regression. |
| Runtime harness command/scenario and exact result | `make build` then `./shipwright --help` / `./shipwright --version` — **blocked**, same network wall (`make build` → `go build -o shipwright .` fails at the identical `go-kit-logger` credential error; confirmed the `-o shipwright` flag itself picked up the renamed `BINARY_NAME` correctly before hitting the wall). Verified by source inspection instead: `cliName = "shipwright"` is passed to `flag.NewFlagSet`, which the `flag` package uses verbatim in its default `Usage()` function's `"Usage of %s:\n"` line (stdlib `flag.FlagSet.defaultUsage`); `versionLogMessage = "Shipwright version"` is the message argument to `showVersion()`'s log call. Both are asserted equal to their expected values in `TestCLIIdentityConstants` and asserted to not contain the old identity tokens. |
| Rollback boundary | Revert `main.go`, `main_test.go`, `Makefile`, `.goreleaser.yml` on this branch (4 files, 27 insertions / 7 deletions, `git diff --stat`) — independent of PR1's import-path rename; reverting PR2 alone restores the `syntegrity-dagger` binary name/flagset/log-message/BINARY_NAME/goreleaser-binary identity without touching module path or imports. |

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `main.go` | Modified | Added `cliName`/`versionLogMessage`/`initLogMessage` constants; `flag.NewFlagSet("syntegrity-dagger", ...)` → `flag.NewFlagSet(cliName, ...)`; `"Syntegrity Dagger initialized successfully"` → `initLogMessage`; `"Syntegrity Dagger version"` → `versionLogMessage`; CI-warning help example `"syntegrity-dagger --pipeline ..."` → `cliName+" --pipeline ..."`. Left untouched (Phase 3 scope, task 3.5): the two `.syntegrity-dagger.yml` `-config` flag default occurrences (lines ~79, ~170). Left untouched (deny-list): `go-kit-logger` import. |
| `main_test.go` | Modified | `os.Args[0]` literal in `TestMain_ErrorOutput` changed to `cliName`; added `TestCLIIdentityConstants` asserting the 3 new constants' values and absence of old-identity substrings. |
| `Makefile` | Modified | `BINARY_NAME=syntegrity-dagger` → `BINARY_NAME=shipwright` (single-line change; all `$(BINARY_NAME)` usages elsewhere in the file pick it up automatically — no other lines touched). |
| `.goreleaser.yml` | Modified | `binary: syntegrity-dagger` → `binary: shipwright` under `builds:`. `owner: getsyntegrity`, `name: syntegrity-dagger` (under `release.github`), and all install-URL curl templates were deliberately left untouched — explicitly deferred to Phase 4 task 4.4 per tasks.md ("Update `.goreleaser.yml` install-URL templates ... if not covered in 2.2"). |

### Gate Status (2.3 / 2.4)

- **2.3 `make build` succeeds**: **blocked by the documented environment constraint**, not a new failure. `make build` → `go build -o shipwright .` fails with the identical pre-existing `go-kit-logger` credential error confirmed at baseline (PR1, `develop`) and again here before and after this change. The `-o shipwright` argument confirms `BINARY_NAME` was picked up correctly. `gofmt -l main.go main_test.go Makefile .goreleaser.yml` → clean (only Go files checked by gofmt; Makefile/goreleaser.yml are not Go — verified by direct read/diff instead).
- **2.4 `./shipwright --help`/`--version` contain no `syntegrity-dagger` token**: verified by **source inspection**, not runtime execution (binary cannot be built here). `rg -n syntegrity main.go` after the GREEN edit shows exactly 3 remaining hits: the deny-listed `go-kit-logger` import and the two Phase-3 `-config` default literals — zero CLI-facing (`--help`/`--version`) occurrences remain.

### Known environment limitation (unchanged from PR1, reconfirmed here)

Reconfirmed identically for this work unit: `go build .`, `go vet .`, and `make build` all fail at `main.go:11:2` (`github.com/getsyntegrity/go-kit-logger` import) with the same `git ls-remote` credential error, both before and after this PR's edits. This is the same sandbox limitation documented in PR1's record above, not a regression introduced by Phase 2. **Final `go build ./...`, `go test -race ./...`, and `./shipwright --help`/`--version` runtime verification must happen in an environment with real credentials (the user's machine or CI) before this PR merges.**

### Deviations from Design

None — implementation matches design.md's token map (rule 7: `Syntegrity Dagger` → `Shipwright`) and the identity-resolution chain (`Makefile BINARY_NAME → .goreleaser.yml → release.yml artifact names`, `main.go flagset/help/version → docs + examples CLI snippets`). Extracting `cliName`/`versionLogMessage`/`initLogMessage` as named constants (instead of inline literal substitution) was a minimal, behavior-preserving refactor to make the rename directly unit-testable per strict-tdd.md's "Extract-Before-Mock" pattern — not a deviation from the design's scope.

### Issues Found

None.

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 3 | Config & env identity (`EnvPrefix`, `yaml_parser.go` 6-entry candidate list, `main.go -config` default) — RED-first | PR 3 |
| 4 | CI/CD & release (`.github/actions/syntegrity-dagger` → `shipwright`, workflows, `dependabot.yml`, `CODEOWNERS`, `.goreleaser.yml` install-URL templates/`owner`/`name`) | PR 4 |
| 5 | Docs & examples (`README.md`, 21 `docs/` files, `CHANGELOG.md`, `examples/**`) | PR 5 |
| 6 | Comments, fixtures, manual edits (`errors.go`/`appconf.test.go` Spanish comments, `.serena/project.yml`, `openspec/config.yaml`, `internal/app/health.go` User-Agent string) | PR 6 |
| 7 | Residual sweep | PR 7 |
| 8 | Final build + test verification | PR 7 |

## Work Unit 3 (PR3) — Config & Env (Phase 3)

Status: **done**
Branch: `feature/shipwright-rebrand-pr3-config-env` (local, not pushed;
stacked on `feature/shipwright-rebrand-pr2-cli-build` per `stacked-to-main`
chain strategy)
Commit: `11b28bf` — `feat(config): rename env prefix and config filenames to shipwright`

### Mode

Strict TDD — RED-first per tasks.md Phase 3 header
(`[RED-first: EnvPrefix, config discovery]`).

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1/3.3 | `config_test.go::TestEnvPrefix` (new) | Unit | N/A — new test, asserts a package-level constant, no existing behavior at risk | ✅ Written asserting `EnvPrefix == "SHIPWRIGHT_"` while the constant was still `"SYNTEGRITY_DAGGER_"` — confirmed failing by direct comparison (`assert.Equal` would report `"SHIPWRIGHT_" != "SYNTEGRITY_DAGGER_"`); execution blocked by the environment constraint below, confirmed via `git stash`/re-run reproducing the identical `go-kit-logger` setup failure on the unmodified baseline | ✅ Confirmed by inspection: after the GREEN edit `EnvPrefix = "SHIPWRIGHT_"` in `config.go` matches the asserted literal exactly (`git diff` + `rg` below) | ➖ Single fixed-value assertion — no additional branch to triangulate for a constant | ➖ None needed |
| 3.1 | `config_test.go::TestNew` (existing, "with environment variables" case) | Unit | N/A — pre-existing test, only its 5 env-var-name literals changed from `SYNTEGRITY_DAGGER_*` to `SHIPWRIGHT_*`; assertions/intent (`wantErr: false`, `DefaultEnvironment` check) untouched | ✅ Same RED as above (shared `EnvPrefix` symbol; the env vars this test sets would silently stop being read once `EnvPrefix` changed if left with old names) | ✅ Confirmed by inspection | ➖ N/A — literal rename only | ➖ None needed |
| 3.2/3.4 | `yaml_parser_test.go::TestYAMLParser_FindConfigFile` + `_Priority` + `_ParentDirectories` (existing, 8 sub-cases across 3 functions) | Unit | N/A — pre-existing tests; only the filename literals changed (`.syntegrity-dagger.yml/.yaml`, `syntegrity-dagger.yml/.yaml`, `.github/syntegrity-dagger.yml`) → shipwright equivalents; assertions/intent (priority order, parent-directory walk, "no config file found" error case) fully preserved | ✅ Written/updated to expect `.shipwright.yml`/`.shipwright.yaml`/`shipwright.yml`/`shipwright.yaml`/`.github/shipwright.yml` while `FindConfigFile()`'s internal candidate list was still `.syntegrity-dagger.*` — guaranteed mismatch (`assert.Equal(t, ".shipwright.yml", filePath)` against a function that could never produce that string) | ✅ Confirmed by inspection: after the GREEN edit to `yaml_parser.go`'s `configFiles` slice, all 6 candidates match the renamed literals `git diff` shows below | ✅ 8 sub-cases across priority order (`.yaml` before `.yml`), current-directory discovery (4 filename variants), `.github/` subdirectory discovery, and parent-directory walk — same coverage shape as before the rename, now proving the new filenames | ➖ None needed — no structural change, pure literal substitution |

### Test Summary

- **Total tests written**: 1 new (`TestEnvPrefix`)
- **Total tests updated (literals only, intent preserved)**: `TestNew`'s "with
  environment variables" case (5 env-var names), `TestYAMLParser_FindConfigFile`
  (5 sub-cases' setup/expected literals), `TestYAMLParser_FindConfigFile_Priority`,
  `TestYAMLParser_FindConfigFile_ParentDirectories`
- **Total tests passing**: Not executable in this sandbox (see Environment
  Constraint below). Verified GREEN by structural/compile-time reasoning:
  `EnvPrefix` and the 6-entry `configFiles` candidate list in `yaml_parser.go`
  match the asserted values exactly (`git diff` + `rg` confirmation below), so
  both `TestEnvPrefix` and `TestYAMLParser_FindConfigFile*` would pass if
  executed with real credentials.
- **Layers used**: Unit (4 test functions touched/added)
- **Approval tests**: None
- **Pure functions created**: 0 (constant + slice-literal rename only; no new
  functions)

### Environment Constraint (traced further this unit)

Same pre-existing sandbox limitation as PR1/PR2 (no git credentials for
`github.com/getsyntegrity/go-kit-logger`), but this unit traced the *exact*
transitive path for `internal/config` specifically: `go list -deps
./internal/config/...` (non-test) succeeds cleanly with zero `go-kit-logger`
or `internal/pipelines/shared` dependency. `go list -deps -test
./internal/config/...` fails — the culprit is `yaml_parser_test.go`'s import
of `github.com/pablogore/shipwright/mocks`, which transitively pulls in
`internal/pipelines/shared/builder.go` → `go-kit-logger`. Confirmed via
`git stash` that `go test ./internal/config/...` fails with the **identical**
`git ls-remote ... terminal prompts disabled` error on the unmodified branch
tip (commit `25d5539`, before any Phase 3 edit) — not a regression introduced
by this work unit. `go build ./internal/config/...` (non-test) succeeds
cleanly both before and after this unit's edits.

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/config/...` — **blocked**: identical pre-existing `go-kit-logger` credential wall (`internal/pipelines/shared/builder.go:10:2: ... git ls-remote ... terminal prompts disabled`), reproduced byte-identical via `git stash` against the unmodified `25d5539` tip. Structural substitute run instead: `go build ./internal/config/...` → exit 0 (clean compile with the new `EnvPrefix` and `configFiles` values); `gofmt -l` on all 5 touched files → only `internal/config/yaml_parser.go` flagged, confirmed via `git diff` to be the same pre-existing, unrelated struct-tag-alignment violation documented in PR1 (not introduced by this unit's 2 changed lines in that file); targeted `rg -n "SYNTEGRITY_DAGGER_\|\.syntegrity-dagger\."` over all `.go` files → zero hits. |
| Runtime harness command/scenario and exact result | CLI run with `SHIPWRIGHT_TOKEN` set + `.shipwright.yml` present — **blocked**, same network wall as PR2 (binary cannot be built: `main.go` imports `go-kit-logger` directly). Verified by source inspection instead: `config.go`'s `New()` builds `env.Provider(EnvPrefix, "_", ...)` where `EnvPrefix = "SHIPWRIGHT_"`, so `SHIPWRIGHT_TOKEN` (and all `SHIPWRIGHT_*` vars) would be read, while `SYNTEGRITY_DAGGER_TOKEN` would not; `yaml_parser.go`'s `FindConfigFile()` checks `.shipwright.yml` first in its `configFiles` slice, so a present `.shipwright.yml` would be discovered. Full runtime verification deferred to an environment with real credentials (user machine or CI), consistent with PR1/PR2. |
| Rollback boundary | Revert `internal/config/config.go`, `internal/config/config_test.go`, `internal/config/yaml_parser.go`, `internal/config/yaml_parser_test.go`, `main.go` on this branch (5 files, 53 insertions / 49 deletions, `git diff --stat` against PR2's tip) — independent of PR1's import-path rename and PR2's CLI/build identity rename; reverting PR3 alone restores `EnvPrefix = "SYNTEGRITY_DAGGER_"` and the `.syntegrity-dagger.*` config discovery candidates without touching the module path, binary name, or flagset identity. |

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/config/config.go` | Modified | `EnvPrefix = "SYNTEGRITY_DAGGER_"` → `EnvPrefix = "SHIPWRIGHT_"`; updated the adjacent doc comment `// 1. Environment variables (SYNTEGRITY_DAGGER_*)` → `(SHIPWRIGHT_*)` to keep it accurate for the same identity surface (not a separate Phase 6 catch-all edit — it directly documents the renamed constant). |
| `internal/config/config_test.go` | Modified | Added `TestEnvPrefix` asserting `EnvPrefix == "SHIPWRIGHT_"`; renamed the 5 `SYNTEGRITY_DAGGER_*` env-var-name literals in `TestNew`'s "with environment variables" case to `SHIPWRIGHT_*` (assertions/intent unchanged). |
| `internal/config/yaml_parser.go` | Modified | `configFiles` slice's 6 candidates (`.syntegrity-dagger.yml/.yaml`, `syntegrity-dagger.yml/.yaml`, `.github/syntegrity-dagger.yml/.yaml`) → shipwright equivalents; doc comment on `YAMLConfig` (`.syntegrity-dagger.yml file` → `.shipwright.yml file`). Import at line 9 confirmed already correct from PR1 (`github.com/pablogore/shipwright/internal/interfaces`) — not re-edited. |
| `internal/config/yaml_parser_test.go` | Modified | All 25 `syntegrity-dagger` filename-literal occurrences (setup/cleanup/expected-value strings across `TestYAMLParser_FindConfigFile`, `_Priority`, `_ParentDirectories`, and the "no config file found" restore-fixture YAML content string) → `shipwright` equivalents via a scoped, whole-file literal substitution (every occurrence in this file was exactly the safe filename token, confirmed via `rg` before editing — no comments referencing the product name in prose needed rule-7 handling here). |
| `main.go` | Modified | `-config` flag default: `flagSet.StringVar(&flags.configFile, "config", ".syntegrity-dagger.yml", ...)` → `".shipwright.yml"`; matching check `if flags.configFile != ".syntegrity-dagger.yml"` → `!= ".shipwright.yml"`. Both are the only two Phase-3-scoped lines left untouched by PR2 (explicitly deferred there). |

### Gate Status (3.6)

`go test ./internal/config/...` — **blocked by the documented environment
constraint**, not a new failure; reproduced identical on the unmodified PR2
tip via `git stash`. Structural verification performed instead: `go build
./internal/config/...` exits 0; `gofmt -l` clean except the pre-existing,
unrelated `yaml_parser.go` violation (confirmed via `git diff` to be outside
this unit's 2 changed lines in that file); `git diff` over `dagger.io/dagger`
and `go-kit-logger` import lines is empty (deny-list non-regression); `rg -n
"SYNTEGRITY_DAGGER_|\.syntegrity-dagger\.|syntegrity-dagger\.y" --glob '*.go'`
returns zero hits repository-wide, confirming Phase 3's Go-code scope is
fully clean. Remaining `SYNTEGRITY_DAGGER_*`/`.syntegrity-dagger.yml` hits
found in `docs/`, `examples/`, and `.github/dependabot.yml`-adjacent docs
belong to Phase 4/5, explicitly out of scope for this unit.

### Deviations from Design

None — implementation matches design.md's token map (rule 4: config-file
literals; rule 5: `SYNTEGRITY_DAGGER_` → `SHIPWRIGHT_`) and the identity
resolution chain (`config.EnvPrefix → SHIPWRIGHT_* → dependabot secret + CI
env`, `yaml_parser candidate list → .shipwright.yml → docs/examples
configs`). The `EnvPrefix` doc-comment edit in `config.go` was a minimal,
same-surface consistency fix (not a scope expansion) since it directly
documents the constant renamed in task 3.3.

### Issues Found

None.

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 4 | CI/CD & release (`.github/actions/syntegrity-dagger` → `shipwright`, workflows, `dependabot.yml`, `CODEOWNERS`, `.goreleaser.yml` install-URL templates/`owner`/`name`) | PR 4 |
| 5 | Docs & examples (`README.md`, 21 `docs/` files, `CHANGELOG.md`, `examples/**`) | PR 5 |
| 6 | Comments, fixtures, manual edits (`errors.go`/`appconf.test.go` Spanish comments, `.serena/project.yml`, `openspec/config.yaml`, `internal/app/health.go` User-Agent string) | PR 6 |
| 7 | Residual sweep | PR 7 |
| 8 | Final build + test verification | PR 7 |

## Work Unit 4 (PR4) — CI/CD & Release (Phase 4)

Status: **done**
Branch: `feature/shipwright-rebrand-pr4-ci-release` (local, not pushed;
stacked on `feature/shipwright-rebrand-pr3-config-env` per `stacked-to-main`
chain strategy)

### Mode

Standard (Phase 4 is not RED-first per tasks.md — pure YAML/text identity
substitution, no testable Go behavior surface).

### ⚠️ OUT-OF-BAND OWNER ACTION REQUIRED BEFORE MERGE (task 4.3)

**`SHIPWRIGHT_TOKEN` must be created as a GitHub repository secret
(Settings → Secrets and variables → Actions, and also under Dependabot
secrets for `dependabot.yml`'s `registries.github-ep.password`) before this
PR merges and before Dependabot next runs against these workflows.**
`ci.yml`, `release.yml`, and `dependabot.yml` now reference
`secrets.SHIPWRIGHT_TOKEN` exclusively — the old `SYNTEGRITY_DAGGER_TOKEN`
secret is no longer read anywhere. This is a GitHub repository-settings
action, not a code task, and cannot be performed from this sandbox. Until
the secret is created, private-module (`go-kit-logger`) authentication in CI
and Dependabot's private registry auth will fail with an empty/missing
token (workflows do have a `GITHUB_TOKEN` fallback for the CI steps, but
`dependabot.yml`'s registry `password` field has no fallback).

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `.github/actions/syntegrity-dagger/` → `.github/actions/shipwright/` | Renamed (`git mv`) | Directory rename per task 4.1. |
| `.github/actions/shipwright/action.yml` | Modified | Applied the full ordered token map (rules 1, 4, 7, 10): `name`/`description` (`Syntegrity Dagger` → `Shipwright`), `.syntegrity-dagger.yml` config default → `.shipwright.yml`, all `syntegrity-dagger` binary/cache/URL/log-message literals → `shipwright`, install URL → `https://github.com/pablogore/shipwright/releases/...`. Zero residual `syntegrity` hits confirmed via `rg`. |
| `.github/workflows/ci.yml` | Modified | Every `SYNTEGRITY_DAGGER_TOKEN` (secret name, env var, echo/error messages) → `SHIPWRIGHT_TOKEN`, repository-wide (11 job-step blocks, ~100+ literal occurrences). Deliberately left untouched (deny-list): `github.com/getsyntegrity/go-kit-logger` references and `GOPRIVATE`/`GONOPROXY`/`GONOSUMDB=github.com/getsyntegrity/*,gitlab.com/syntegrity/*` — these name the real company org/private module, not the product identity. |
| `.github/workflows/release.yml` | Modified | `BINARY_NAME: 'syntegrity-dagger'` → `'shipwright'`; all `SYNTEGRITY_DAGGER_TOKEN` occurrences → `SHIPWRIGHT_TOKEN`; the per-OS/arch tar/zip extraction block's `syntegrity-dagger_*` glob patterns and `raw-binaries/syntegrity-dagger-*` output filenames → `shipwright` equivalents (matches PR2's `.goreleaser.yml` `binary: shipwright` archive-name template). Deny-listed `getsyntegrity`/`gitlab.com/syntegrity` GOPRIVATE lines left untouched. |
| `.github/dependabot.yml` | Modified | Comment + `password: ${{ secrets.SYNTEGRITY_DAGGER_TOKEN }}` → `SHIPWRIGHT_TOKEN`. `exclude-patterns: ["github.com/getsyntegrity/*"]` (×2) deliberately left untouched — deny-listed company-org exclusion pattern, unrelated to the product name. |
| `.github/CODEOWNERS` | Modified | Header comment `# Code owners for syntegrity-dagger` → `# Code owners for shipwright` (single line; ownership rules `* @pablonqn`, `docs/`, `scripts/`, etc. unchanged). |
| `.github/rulesets/README.md` | Modified | Applied token map; additionally hand-corrected two `gh api repos/getsyntegrity/...` / `https://api.github.com/repos/getsyntegrity/...` lines to `repos/pablogore/...` — the catch-all rule alone would have left `getsyntegrity/shipwright` (wrong owner) since design rule 3 (`getsyntegrity/syntegrity-dagger` → `pablogore/shipwright`, bare form) didn't literal-match these `repos/`-prefixed strings. Verified against design.md's decided repo owner (`pablogore`, matching `go.mod`/`.goreleaser.yml`). |
| `.goreleaser.yml` | Modified | Applied full token map to the header comment, `release.github.owner`/`name` (`getsyntegrity`/`syntegrity-dagger` → `pablogore`/`shipwright`), the install-URL curl templates in the release footer (both direct-binary and archive-download variants), the "Download Shipwright" CI/CD usage snippet, and the commented-out (disabled) Docker `image_templates`/OCI label URLs. `binary: shipwright` (already set by PR2) left untouched. This closes task 4.4's explicit follow-up: PR2 only touched the `binary:` field and deliberately deferred `owner:`/`name:`/URL templates here. |
| `scripts/apply-branch-protection.sh` | Modified | `REPO="getsyntegrity/syntegrity-dagger"` → `REPO="pablogore/shipwright"`. Not explicitly named in the task's file list, but included per design.md's Execution Phases table (Phase 4 row explicitly lists `scripts/`) — this script is the "Opción 1: Script Automatizado" companion to `.github/rulesets/README.md`, which was in scope; leaving it stale would have produced an inconsistent repo-name reference right next to the corrected doc. |
| `.gitignore` | Modified | `coverage/*syntegrity-dagger` → `coverage/*shipwright`. Also included per design.md's Phase 4 scope row (`.gitignore` explicitly listed) — a coverage-artifact ignore pattern tied to the renamed binary name. |

### Deliberately NOT touched (confirmed in scope for Phase 5, not Phase 4)

`examples/github-actions/service-ci-example.yml` (7 occurrences) references
`uses: ./.github/actions/syntegrity-dagger` — this path now points at a
directory that no longer exists after this PR's `git mv`, but the file lives
under `examples/**`, which is explicitly Phase 5 scope per tasks.md and the
orchestrator's instructions ("Do NOT touch Phase 5+"). Flagging this as a
**known, intentional, self-resolving inconsistency**: PR5 (already scheduled
to update all of `examples/**`) will fix this reference. No `.github/workflows/*.yml`
file self-references the composite action directory (confirmed via `rg`), so
this PR does not break CI.

### Gate Status (4.5)

`yamllint` **was not pre-installed** in this sandbox; installed via
`brew install yamllint` (1.38.0 — the repo's `.pre-commit-config.yaml` pins
`v1.35.1`, close enough for a structural gate check, not an exact
reproduction of the CI pre-commit hook version). Ran
`yamllint -c .yamllint` against all 5 changed YAML files
(`ci.yml`, `release.yml`, `dependabot.yml`, `action.yml`, `.goreleaser.yml`).

Result: **no new lint errors introduced**. Confirmed via before/after
comparison — reverted the rename with `git stash` + a temporary reverse
`git mv`, re-ran yamllint against the unmodified PR3-tip files, and diffed
rule-type counts:

| | Before (baseline) | After (this PR) |
|---|---|---|
| Errors (total) | 297 | 297 |
| Error rule types | `empty-lines`(2), `indentation`(13), `new-line-at-end-of-file`(1), `quoted-strings`(281) | identical: `empty-lines`(2), `indentation`(13), `new-line-at-end-of-file`(1), `quoted-strings`(281) |
| Warnings (total) | 262 | 244 |

All 297 errors are pre-existing baseline violations (the repo's YAML never
conformed to `extends: default`'s quoted-strings/indentation rules — a
pre-existing condition, not introduced by this rename). The warning count
*dropped* (262 → 244) because renamed literals (`shipwright` is shorter than
`syntegrity-dagger`) pushed some lines under the 120-char `line-length`
warning threshold — a net improvement, not a regression. Zero new error or
warning categories appeared after the rename.

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | No Go test surface in this work unit (pure YAML/shell-in-YAML/markdown text). Structural gate substituted: `yamllint -c .yamllint` on all 5 changed YAML files — see Gate Status table above (297/297 pre-existing errors, 244 vs 262 warnings, zero new categories). |
| Runtime harness command/scenario and exact result | N/A — no compiled binary or executable code path changes in this work unit; GitHub Actions workflow YAML cannot be executed locally without `act` (not installed, out of scope to install for a rename-only PR). Verified by structural inspection instead: `rg -in syntegrity` on every touched file returns zero hits except the confirmed deny-listed `getsyntegrity`/`gitlab.com/syntegrity` company-org and `go-kit-logger` private-module references (spot-checked line-by-line above). |
| Rollback boundary | Revert the 9 files in this branch (`.github/CODEOWNERS`, `.github/actions/{syntegrity-dagger→shipwright}/action.yml`, `.github/dependabot.yml`, `.github/rulesets/README.md`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.gitignore`, `.goreleaser.yml`, `scripts/apply-branch-protection.sh`; 201 insertions / 201 deletions, `git diff --stat`) — independent of PR1's import rename, PR2's CLI/build identity, and PR3's config/env identity; reverting PR4 alone restores the `syntegrity-dagger` composite-action path, workflow token names, CODEOWNERS header, rulesets doc, goreleaser owner/name/URLs, and branch-protection script repo reference without touching module path, binary name, flagset, or config discovery. |

### Deviations from Design

None on the core token-map application. Two scope clarifications, both
consistent with design.md rather than deviations from it:

1. Included `scripts/apply-branch-protection.sh` and `.gitignore` — not
   named in the orchestrator's literal task-4.2 file list, but explicitly
   listed in design.md's Execution Phases table row for Phase 4
   (`git mv .github/actions/syntegrity-dagger shipwright; ci.yml, release.yml,
   dependabot.yml, CODEOWNERS, rulesets/README.md, scripts/, .gitignore`).
   Both are direct companions to files already in scope (the branch-protection
   script is `rulesets/README.md`'s "Opción 1" automation; the `.gitignore`
   line is the coverage-artifact pattern for the renamed binary).
2. `.github/rulesets/README.md`'s two `repos/getsyntegrity/...` API-path
   lines needed a manual correction beyond the literal token map (see Files
   Changed table) — the catch-all rule 10 alone would have produced
   `getsyntegrity/shipwright` (wrong, mixed old-org/new-name), not
   `pablogore/shipwright`. Corrected using design rule 3's intent
   (`getsyntegrity/syntegrity-dagger` → `pablogore/shipwright`) since the
   literal bare-form rule didn't match the `repos/`-prefixed string. Applied
   the same correction to `.goreleaser.yml`'s `release.github.owner` field,
   which had the identical stale-org pattern.

### Issues Found

- `examples/github-actions/service-ci-example.yml` will have a dangling
  `uses: ./.github/actions/syntegrity-dagger` reference until PR5 updates
  `examples/**` — flagged above under "Deliberately NOT touched", not fixed
  here per explicit Phase 5 scope boundary. No workflow in `.github/workflows/`
  is affected (none self-reference the composite action).
- Review-budget note: this work unit's diff is **201 insertions / 201
  deletions = 402 changed lines**, marginally over the 400-line guard
  (0.5% over). Flagging rather than silently absorbing: the entire diff is
  1:1 mechanical literal-token substitution across pre-scoped files (no new
  logic, no structural change), matching the same low-risk profile as PR1-3.
  Given Phase 4 was already assigned as one complete, atomic work-unit
  boundary (composite-action rename + all its dependent CI/CD identity
  references form a single coherent, indivisible deliverable — splitting the
  action-directory rename from its own `uses:`-adjacent workflow references
  would leave an inconsistent intermediate state), this was implemented as
  the single assigned PR4 slice rather than further sub-split. No further
  action taken without explicit maintainer direction; **surfacing this
  border-line size for visibility**, not requesting a decision that blocks
  progress, since the work-unit boundary itself was pre-assigned.

### Known environment limitation (unchanged from PR1-3, N/A for this unit's gate)

The `go-kit-logger` sandbox credential wall documented in PR1/PR2/PR3 does
not affect this work unit's gate (`yamllint`, not `go build`/`go test`) —
noted here only for completeness of the cumulative record. `ci.yml` and
`release.yml`'s references to `github.com/getsyntegrity/go-kit-logger`
remain deliberately untouched (deny-list).

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 5 | Docs & examples (`README.md`, 21 `docs/` files, `CHANGELOG.md`, `examples/**` — including the dangling composite-action `uses:` path noted above) | PR 5 |
| 6 | Comments, fixtures, manual edits (`errors.go`/`appconf.test.go` Spanish comments, `.serena/project.yml`, `openspec/config.yaml`, `internal/app/health.go` User-Agent string) | PR 6 |
| 7 | Residual sweep | PR 7 |
| 8 | Final build + test verification | PR 7 |

## Work Unit 5 (PR5) — Docs & Examples (Phase 5)

Status: **done**
Branch: `feature/shipwright-rebrand-pr5-docs-examples` (local, not pushed;
stacked on `feature/shipwright-rebrand-pr4-ci-release` per `stacked-to-main`
chain strategy)
Commit: `46ed0ab` — `docs: rename docs and examples identity to shipwright`

### Mode

Standard (Phase 5 is not RED-first per tasks.md — pure prose/YAML/shell/
groovy identity substitution, no testable Go behavior surface; consistent
with design.md's TDD-ordering decision that docs are not testable).

### Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `README.md` | Modified | Title, badge URL, install curl URLs/binary names, `--config` example, GitLab/Jenkins CI snippets, repo clone URL, issues/discussions links — full token map applied. Also fixed a pre-existing `SYNTERGRITY_VERSION` (misspelled) example env var to `SHIPWRIGHT_VERSION` for identity consistency (same GitLab CI snippet already being rewritten for the URL). |
| `CHANGELOG.md` | Modified | Header sentence, `[0.0.1]` initial-release entry, and the Links section's GitHub Releases/Documentation URLs — all 4 historical `syntegrity-dagger`/`Syntegrity Dagger` occurrences rewritten to Shipwright per task 5.3. |
| `docs/*.md` (21 files) | Modified | Full ordered token map (rules 1, 3, 5–10) applied to every file: `ANALYSIS_AND_RECOMMENDATIONS.md`, `API.md`, `ARCHITECTURE.md`, `BRANCHING_STRATEGY.md`, `BRANCH_PROTECTION_RULES.md`, `CONFIGURATION.md`, `CONFLICT_PREVENTION_GUIDE.md`, `DEPENDABOT_SETUP.md`, `DEPLOYMENT_ENVIRONMENTS.md`, `GITHUB_FREE_ACCOUNT_SETUP.md`, `INTEGRATION_GUIDE.md`, `LOCAL_USAGE.md`, `MERGE_CONFLICT_RESOLUTION.md`, `PIPELINE_DEVELOPMENT.md`, `PLUGINS.md`, `PRODUCTION_DEPLOYMENT.md`, `README.md`, `RELEASE_PROCESS.md`, `REQUIRE_CI_SUCCESS.md`, `WORKFLOW_DIAGRAMS.md`, `GITHUB_RULESETS.md` (no `syntegrity[-_]?dagger` hits — confirmed zero diff, left untouched as a no-op). Covers binary name, module import path snippets, config filename (`.syntegrity-dagger.yml` → `.shipwright.yml` and its `.local`/`.onpremise` variants), env-var prefix (`SYNTEGRITY_DAGGER_*` → `SHIPWRIGHT_*`), and product-name prose. |
| `docs/REQUIRE_CI_SUCCESS.md` | Modified (manual correction) | `gh api repos/getsyntegrity/syntegrity-dagger/rulesets` → `repos/pablogore/shipwright/rulesets` (catch-all rule 10 alone would have left the wrong org; corrected using the same `getsyntegrity/<product>` → `pablogore/shipwright` bare-path logic PR4 established for `rulesets/README.md`). The file's two `orgs/getsyntegrity/rulesets` org-level (not repo-scoped) API examples were deliberately left untouched — that endpoint targets the entire `getsyntegrity` GitHub org, a company-identity reference with no product-name segment, matching the deny-list. |
| `docs/ARCHITECTURE.md` | Modified (manual correction) | Beyond the token map (title, prose, diagram titles, all `Syntegrity Dagger` → `Shipwright`), the bare C4 diagram node identifier `syntegrity` (used in `System(syntegrity, ...)`, `Container_Boundary(syntegrity, ...)`, and 6 `Rel(syntegrity, ...)` lines) was renamed to `shipwright`. This id refers to this product itself (not the company), so leaving it would have left a residual bare `syntegrity` hit that does not belong to the company-identity deny-list, violating the spec's "Company identity left untouched" scenario (every bare `syntegrity` match must belong to the real company). |
| `examples/configs/production.yml` | Modified | Header comment + `--health` example command — token map applied. |
| `examples/github-actions/{go-kit-ci,tenant-svc-ci}.yml` | Modified | Full token map plus a manual correction: both files' release-download `URL=` lines used `github.com/syntegrity/syntegrity-dagger` (a pre-existing org-name variant/typo, distinct from the real `getsyntegrity` org) — the catch-all rule alone would have produced `github.com/syntegrity/shipwright` (an org path that doesn't exist under either the real company org or the product's actual owner). Corrected to `github.com/pablogore/shipwright` for consistency with every other install-URL example in the repository (README.md, RELEASE_PROCESS.md, etc.). |
| `examples/github-actions/service-ci-example.yml` | Modified | All 7 `uses: ./.github/actions/syntegrity-dagger` composite-action references → `./.github/actions/shipwright` via the catch-all rule — resolves the dangling reference flagged in PR4's apply-progress record (PR4 renamed the real directory but explicitly deferred this file to Phase 5). Also 7× `config: .syntegrity-dagger.yml` → `.shipwright.yml`. |
| `examples/local/{README.md,local-ci.sh,run-local.sh}` | Modified | Token map applied to prose, `CONFIG_FILE` default literal, install/download command examples, and the two `.syntegrity-dagger.{dev,staging}.yml` variant examples in `README.md`. |
| `examples/on-premise/jenkins-pipeline.groovy` | Modified | Token map applied across the Groovy pipeline: `SYNTEGRITY_DAGGER_VERSION` env var, `--config .syntegrity-dagger.yml` (×6 stages), install-URL curl command, binary name. |

### Deliberately NOT touched (deny-list, confirmed byte-identical via `git diff`)

- `examples/configs/tenant-svc.yml` — zero diff; its `ghcr.io/syntegrity` registry namespace is the real company's registry, not product identity, per explicit instruction and design.md's deny-list.
- `examples/configs/{go-service,go-service-with-nomad}.yml` — zero `syntegrity` hits found; not modified.
- `examples/{complete_usage_example.go,dynamic_pipeline_example.go}` — zero `syntegrity` hits (PR1 already covered `examples/` for the module-path import rewrite); `gofmt -l` confirmed clean, no diff.
- `docs/DEPENDABOT_SETUP.md`, `docs/GITHUB_FREE_ACCOUNT_SETUP.md` — `github.com/getsyntegrity/go-kit-logger` references (4 occurrences total) left byte-identical; deny-listed private-module identity.
- `docs/ANALYSIS_AND_RECOMMENDATIONS.md` — 3 bare-`syntegrity` occurrences deliberately preserved: "servicios Go de Syntegrity" (company-identity prose, not the product), `https://plugins.syntegrity.io/` and `syntegrity-plugin-registry` (hypothetical company-hosted example registry, same category as `ghcr.io/syntegrity`), and `.syntegrity-pipeline.yml` (an illustrative anti-pattern filename example — no `dagger` token pairing, not this product's actual config filename).
- `AGENTS.md`, `Makefile`'s `gitlab.com/syntegrity` grep, root `1export`, `internal/pipelines/shared/{ssh,https}_cloner.go`, `internal/pipelines/infra/` — not opened for edit; outside Phase 5 scope, confirmed no diff.

### Gate Status (5.5 — link check)

No link-checking tool (`markdown-link-check`, `lychee`) was pre-installed in
this sandbox; a manual `rg`-based check was performed instead, per the task's
explicit fallback instruction:

1. Confirmed zero remaining references to the pre-rename composite-action
   path (`rg -n '\.github/actions/syntegrity'` across `README.md`,
   `CHANGELOG.md`, `docs/`, `examples/` → zero hits).
2. Confirmed `.github/actions/shipwright/` exists on disk and
   `.github/actions/syntegrity-dagger/` is correctly absent (PR4's `git mv`).
3. Extracted every relative (non-`http`) markdown link in `README.md` and
   `docs/README.md` — none reference any renamed path or filename; the
   `docs/README.md` links to `QUICK_START.md`, `INSTALLATION.md`,
   `PIPELINES/GO_KIT.md`, etc. are **pre-existing broken links to files that
   were never created** (confirmed present before this PR's changes via
   `git diff docs/README.md` showing only content-line edits, no link-target
   changes) — a pre-existing documentation debt unrelated to this rename, out
   of this gate's scope (the gate covers links broken *by* the rename, not
   pre-existing dead links).

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | No Go test surface in this work unit (pure markdown/YAML/shell/groovy text). Structural gate substituted: `rg -ic 'syntegrity[-_ ]?dagger' README.md CHANGELOG.md docs examples` → zero hits (full residual sweep clean for Phase 5 scope); `gofmt -l` on the two untouched `examples/*.go` files → clean (zero diff, confirming no accidental collateral edit). |
| Runtime harness command/scenario and exact result | N/A — no compiled binary or executable code path changes in this work unit; markdown/YAML/shell/Groovy example files are documentation artifacts, not executed in this sandbox or CI directly. Verified by structural inspection instead: manual link check (Gate 5.5 above) and line-by-line `git diff` review of every changed file confirming only identity-token substitutions, no structural/semantic changes to example commands. |
| Rollback boundary | Revert the 30 files on this branch (`README.md`, `CHANGELOG.md`, 21 `docs/*.md` files, 8 `examples/**` files; 521 insertions / 521 deletions, `git diff --stat`) — independent of PR1's import rename, PR2's CLI/build identity, PR3's config/env identity, and PR4's CI/CD identity; reverting PR5 alone restores the `syntegrity-dagger` documentation and examples identity without touching module path, binary name, config discovery, or CI/CD workflow files. |

### Deviations from Design

None on the core token-map application. Three scope clarifications beyond
the literal token map, all consistent with design.md's intent and PR4's
established precedent for manual bare-org-path corrections:

1. `docs/REQUIRE_CI_SUCCESS.md`'s `repos/getsyntegrity/syntegrity-dagger` →
   `repos/pablogore/shipwright` (same correction class as PR4's
   `rulesets/README.md` fix).
2. `examples/github-actions/{go-kit-ci,tenant-svc-ci}.yml`'s
   `github.com/syntegrity/syntegrity-dagger` → `github.com/pablogore/
   shipwright` (a pre-existing org-name variant that the literal token map
   would have mapped to a still-wrong, nonexistent `github.com/syntegrity/
   shipwright` path).
3. `docs/ARCHITECTURE.md`'s bare C4 diagram node id `syntegrity` → `shipwright`
   — not part of any literal token-map rule (no `dagger` pairing), but
   required for the spec's "Zero Residual Old-Identity References" /
   "Company identity left untouched" scenario, since this bare token
   unambiguously refers to the product, not the company.

### Issues Found

- Review-budget note: this work unit's diff is **521 insertions / 521
  deletions = 1042 changed lines**, well over the 400-line guard (2.6×).
  Flagging rather than silently absorbing, consistent with PR4's precedent:
  the entire diff is 1:1 mechanical literal-token substitution across
  pre-scoped files (no new logic, no structural change), same low-risk
  profile as PR1–4. Phase 5 was pre-assigned by the orchestrator as one
  complete, atomic work-unit boundary ("ONLY Phase 5 Docs & Examples")
  spanning 30 files that each individually need the identical rename — no
  natural sub-split preserves "clear start, clear finish, autonomous scope"
  better than the phase boundary itself (splitting `docs/` alphabetically or
  by file, for example, would produce arbitrary, harder-to-review slices
  with no independent deliverable value). No further action taken without
  explicit maintainer direction; surfacing this size for visibility, not
  requesting a decision that blocks progress.
- `docs/README.md` contains numerous pre-existing broken relative links
  (`QUICK_START.md`, `INSTALLATION.md`, `PIPELINES/GO_KIT.md`, etc. — files
  that do not exist in the repository). Confirmed pre-existing via `git diff`
  (only content lines changed, not link targets) and explicitly out of this
  PR's Gate 5.5 scope (link check covers rename-induced breakage only). Not
  fixed here; flagging for a future documentation-cleanup follow-up outside
  this SDD change.

### Known environment limitation (unchanged from PR1-4, N/A for this unit's gate)

The `go-kit-logger` sandbox credential wall documented in PR1–3 does not
affect this work unit's gate (manual `rg`-based link/residual check, not
`go build`/`go test`) — noted here only for completeness of the cumulative
record. `docs/DEPENDABOT_SETUP.md` and `docs/GITHUB_FREE_ACCOUNT_SETUP.md`'s
references to `github.com/getsyntegrity/go-kit-logger` remain deliberately
untouched (deny-list).

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 7 | Residual sweep + final build/test verification | PR 7 |

## Work Unit 6 (PR6) — Comments, Fixtures & Manual Edits (Phase 6)

Status: done
Branch: `feature/shipwright-rebrand-pr6-comments-fixtures` (stacked on PR5's
`feature/shipwright-rebrand-pr5-docs-examples` @ `46ed0ab`)
Commit: `a7bcaf6` — `refactor(comments): rename residual comments and literals to shipwright`

### Scope Discovery

Per the task instruction, `rg -ni 'syntegrity[-_ ]?dagger' internal/ tests/`
was run first to find the exact remaining set rather than guessing at file
locations. It returned exactly 3 hits across 2 files — none in `_test.go`,
`mocks*.go`, or `tests/` (Phase 1-5 had already fully covered those buckets):

- `internal/app/app.go:96` — `logger.L().InfoContext(ctx, "Starting Syntegrity Dagger application...")`
- `internal/app/app.go:102` — `logger.L().InfoContext(ctx, "Stopping Syntegrity Dagger application...")`
- `internal/app/health.go:140` — `req.Header.Set("User-Agent", "syntegrity-dagger/1.0")`

A broader `rg -ni 'syntegrity'` (no `dagger` pairing required) across the
same `internal/` and `tests/` scope returned zero additional hits, confirming
no bare-token residuals were missed.

### Files Changed (4, git-tracked)

| File | Change |
|---|---|
| `internal/config/errors.go:1` | Spanish package doc comment "aplicación Syntegrity" → "aplicación Shipwright" (6.2 — manual edit, does not match the `syntegrity[-_ ]?dagger` token-map pattern). |
| `internal/config/appconf.test.go:1` | Identical Spanish comment, same replacement (6.3 — manual edit). |
| `internal/app/app.go:96,102` | Start/stop log messages "Syntegrity Dagger application" → "Shipwright application" (6.4 — catch-all, found via the `rg` sweep). |
| `internal/app/health.go:140` | Outbound HTTP `User-Agent` header literal `syntegrity-dagger/1.0` → `shipwright/1.0` (6.4 — catch-all, found via the `rg` sweep). |

Diff: 5 insertions / 5 deletions across 4 files — smallest work unit in the
chain so far, consistent with Phase 6 being a narrow cleanup pass after
PR1-5 already covered the bulk of the repo.

### Non-git-tracked config edits (6.5)

`openspec/config.yaml` (SDD tooling config, sibling to the deny-listed
`openspec/changes/shipwright-project-rebranding/**` artifact directory, not
under it) — `testing.build_command` (line 25) and `verify.build_command`
(line 65) both changed from `go build -o syntegrity-dagger .` to
`go build -o shipwright .`, per the explicit instruction to keep this field
accurate to the renamed binary.

This file was edited on disk but **not staged into the PR6 git commit**,
consistent with PR1-5 precedent: `git log --oneline -- openspec/` and
`git ls-files openspec/` both confirm nothing under `openspec/` has ever
been committed on this chain — the whole directory (`openspec/`, along with
`.atl/` and `.serena/`) is untracked local SDD/IDE tooling state, not part
of the reviewable code diff.

`.serena/project.yml` was inspected per 6.5's instruction ("if it exists and
references the old build command `-o syntegrity-dagger`"). It exists and
contains `project_name: "syntegrity-dagger"`, but no `-o syntegrity-dagger`
build-command string — the task's stated trigger condition does not match,
so it was left unchanged. Flagging `project_name` as a possible residual
identity string outside this task's literal scope (see Issues Found below).

### Gate Status (6.6 — `gofmt -l .`)

`gofmt -l` on the 4 touched `.go` files (`internal/config/errors.go`,
`internal/config/appconf.test.go`, `internal/app/app.go`,
`internal/app/health.go`) returns empty — clean. Full-repo `gofmt -l .`
still reports the same pre-existing baseline violations flagged before this
work unit began (`internal/cache/*`, `internal/config/validation*.go`,
`internal/config/yaml_parser.go`, `internal/config/yaml_step_config*.go`,
`internal/pipelines/go-service/pipeline_test.go`,
`internal/pipelines/test/gotester.go`) — none touched or fixed by this PR,
explicitly out of scope per the task instruction.

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test -race ./...` could not run — same pre-existing sandbox `go-kit-logger` private-module credential wall as PR1-5 (unchanged, non-regression). Structural gate substituted: `rg -ni 'syntegrity[-_ ]?dagger' internal/ tests/` → zero hits post-change (full residual sweep clean for Phase 6 scope); `gofmt -l` on all 4 touched files → clean. |
| Runtime harness command/scenario and exact result | N/A per the work unit's own harness definition — comment/test-literal only, no runtime behavior change. Verified by source inspection: the two `app.go` strings are structured-log message text only (no assertions or control flow reference them); the `health.go` User-Agent header value is sent on an outbound HTTP request whose only consumer is the remote server's access log, confirmed via `rg -n 'User-Agent'` showing this is the sole occurrence in the codebase (no test asserts on the header's literal value). |
| Rollback boundary | Revert the 4 files on this branch (5 insertions / 5 deletions) — independent of PR1's import rename, PR2's CLI/build identity, PR3's config/env identity, PR4's CI/CD identity, and PR5's docs/examples; reverting PR6 alone restores the four comment/log/header strings without touching module path, binary name, config discovery, CI/CD, or documentation. |

### Deviations from Design

None. All 4 files changed were either explicitly named in the task (6.2,
6.3) or surfaced by the exact `rg` sweep the task instructed (6.1/6.4,
covering `internal/app/app.go` and `internal/app/health.go` — consistent
with the prior PR5 apply-progress note flagging `health.go`'s User-Agent
string as expected Phase 6 scope).

### Issues Found

- `.serena/project.yml:2` — `project_name: "syntegrity-dagger"`. Not changed:
  outside 6.5's literal trigger condition (only the build-command string was
  in scope) and outside this task's explicit file list. Flagging for Phase 7
  residual sweep or a follow-up decision, since `.serena/` is untracked local
  tooling metadata, not part of the reviewable git history on this chain.
- Confirmed (not a defect): `docs/DEPENDABOT_SETUP.md` /
  `docs/GITHUB_FREE_ACCOUNT_SETUP.md`'s `github.com/getsyntegrity/go-kit-logger`
  references and all deny-listed company-identity strings remain untouched —
  outside this work unit's file set entirely, no action taken.

### Known environment limitation (unchanged from PR1-5, N/A for this unit's gate)

The `go-kit-logger` sandbox credential wall documented in PR1-3 does not
block this work unit's gate (`rg`-based residual sweep + `gofmt -l` on
touched files, not `go build`/`go test`) — noted here only for completeness
of the cumulative record.

### Remaining phases (not started, out of scope for this run)

| Phase | Scope | PR |
|---|---|---|
| 7 | Residual sweep + final build/test verification (Phase 7-8) | PR 7 |
| 8 | Final build + test verification | PR 7 |

## Work Unit 7 (PR7) — Residual Sweep + Final Build/Test Verification (Phase 7-8)

Status: **done**
Branch: `feature/shipwright-rebrand-pr7-residual-sweep` (local, not pushed;
stacked on `feature/shipwright-rebrand-pr6-comments-fixtures` @ `a7bcaf6`
per `stacked-to-main` chain strategy)
Commit: `7160e46` — `chore: sweep residual syntegrity-dagger identity strings`

### Mode

Standard (Phase 7-8 are verification/sweep tasks, not RED-first — no new
testable Go behavior surface; consistent with design.md's TDD-ordering
decision).

### Task 7.1 — Product-pattern sweep (`syntegrity[-_ ]?dagger`)

Initial run of the exact command from the task (excluding
`openspec/changes/shipwright-project-rebranding/**`, `.git/**`,
`coverage/**`, `.serena/**`) surfaced **4 hits** beyond the two documented
exceptions (Makefile's `gitlab.com/syntegrity` grep — which doesn't even
match this pattern since it lacks "dagger" — and the root `1export`, which
contains zero `syntegrity` occurrences at all):

| File:Line | Content | Fix |
|---|---|---|
| `main_test.go:98` | Comment: `MUST NOT contain any trace of the legacy Syntegrity Dagger identity.` | Reworded to `MUST NOT contain any trace of the pre-rebrand product identity.` (preserves meaning, drops the token; the `NotContains` assertions on lines 103-105 that check for the literal string `"syntegrity"`/`"Syntegrity"` were correctly left untouched — they are the enforcement mechanism itself, not a residual reference) |
| `openspec/specs/README.md:3` | `This directory holds the merged, canonical specs for \`syntegrity-dagger\`...` | `syntegrity-dagger` → `shipwright`. Untracked file (`openspec/` has never been committed on this chain, confirmed via `git ls-files openspec/` returning empty) — edited on disk per the same precedent as PR6's `openspec/config.yaml` edit, not part of any git commit. |
| `Makefile:1` | `# Syntegrity Dagger Makefile` | → `# Shipwright Makefile` |
| `Makefile:39` | `@echo -e "$(BLUE)🚀 Syntegrity Dagger - Makefile$(NC)"` | → `🚀 Shipwright - Makefile` |

Re-run after fixes: **zero hits** (confirmed via `rg ... ; echo "EXIT:$?"` →
exit 1 = no matches). Re-confirmed again after committing, against the final
`HEAD` tree — still zero hits.

### Task 7.2 — Bare-term sweep (`syntegrity`, no `dagger` pairing)

`rg -i syntegrity --glob '!openspec/changes/shipwright-project-rebranding/**' --glob '!.serena/**'`
returned **106 matching lines across 30 files**. Every hit was individually
classified:

| Category | Files / Pattern | Count | Verdict |
|---|---|---|---|
| Deny-listed: `go-kit-logger` import/dependency | `main.go`, `go.mod`, `go.sum`, `AGENTS.md`, `docs/DEPENDABOT_SETUP.md`, `docs/GITHUB_FREE_ACCOUNT_SETUP.md`, and 15 `internal/**` files importing `github.com/getsyntegrity/go-kit-logger/pkg/logger` | ~35 lines | Preserved (deny-list) |
| Deny-listed: `internal/pipelines/infra/` (`SyntegrityInfraPipeline`, `syntegrity-infra`, "infraestructura de Syntegrity" comment) | `internal/pipelines/infra/pipeline.go`, `pipeline_test.go` | ~35 lines | Preserved (deny-list) |
| Deny-listed: `internal/pipelines/shared/{ssh,https}_cloner.go` company strings (`ci@getsyntegrity.com`, `"Syntegrity CI"`, `$HOME/.ssh/syntegrity`) | 2 files | 5 lines | Preserved (deny-list) |
| Deny-listed: Makefile `gitlab.com/syntegrity` coverage-filter grep | `Makefile:185,236,238,239,240,274` | 6 lines | Preserved (deny-list) |
| Deny-listed: `examples/configs/tenant-svc.yml` `ghcr.io/syntegrity` registry namespace | 1 file | 1 line | Preserved (deny-list) |
| Deny-listed: `AGENTS.md` `eventengine` examples | `AGENTS.md:850,1113` | 2 lines | Preserved (deny-list) |
| Company/org reference, not itemized but covered by spec's "Company identity left untouched" scenario: `docs/REQUIRE_CI_SUCCESS.md` org-level `orgs/getsyntegrity/rulesets` API examples | 1 file | 2 lines | Preserved (real `getsyntegrity` GitHub org reference; already reviewed and deliberately left untouched per PR5's apply-progress record) |
| Company-owned example value: `internal/pipelines/go-service/pipeline_test.go` `GitRepo` test fixture `https://github.com/getsyntegrity/example-service.git` | 1 file | 3 lines | Preserved (company-owned example value, matches spec's allowed category) |
| Test literal proving absence (not a residual reference) | `main_test.go:103-105` `assert.NotContains(t, ..., "syntegrity")` | 3 lines | Preserved (this is the enforcement mechanism for the Phase 2 CLI-identity test, not a leftover) |

**Zero unclassified hits.** Every one of the 106 matches falls into a
documented deny-list category, the spec's broader "Company/Org References
(Preserved)" scenario, or a legitimate test-literal proving the old identity
is absent. No fixes were required for 7.2.

### Task 7.3 — Exclusion confirmation

Both sweep commands were run with the exact flag
`--glob '!openspec/changes/shipwright-project-rebranding/**'` (plus 7.1's
additional `.git/**`, `coverage/**`, `.serena/**` excludes) — confirmed by
direct inspection of the commands executed above. SDD artifacts under this
change folder intentionally quote the old name for historical/documentation
purposes and are correctly excluded from both sweeps.

### Task 8.1 — `go build ./...`

**Environment-blocked, needs user/CI verification.** Fails at
`main.go:11:2` with:

```
main.go:11:2: github.com/getsyntegrity/go-kit-logger@v0.0.0-20250828114729-566d9913c10b: invalid version: git ls-remote -q --end-of-options origin in .../vcs/...: exit status 128:
	fatal: could not read Username for 'https://github.com': terminal prompts disabled
```

This is the exact same error signature documented in PR1-6. To rule out any
possibility this is a rename-induced regression rather than a pre-existing
sandbox limitation, a dedicated `git worktree` was created against the
**unmodified baseline commit `b14c726`** (before any of the 7 PRs) and
`go build ./...` was run there — it fails with the byte-identical error
message. Confirmed not a regression. Worktree removed after verification.

### Task 8.2 — `go test -race ./...`

**Environment-blocked, needs user/CI verification.** 13 packages fail with
the identical credential error (`[setup failed]`): root package, `examples`,
`internal/app`, `internal/config`, `internal/executors`,
`internal/pipelines/go-service`, `internal/pipelines/infra`,
`internal/pipelines/shared`, `internal/pipelines/test`, `internal/plugins`,
`mocks`, `tests`, `tests/mocks`. The 3 packages with no transitive
`go-kit-logger` dependency pass cleanly: `internal/cache`,
`internal/interfaces`, `internal/pipelines` (all `ok`), consistent with
PR1's original finding. `internal/pipelines/common` has no test files.

No new or different error signature appeared anywhere in the output — every
failure is the identical `git ls-remote ... terminal prompts disabled`
credential error. **Coverage thresholds (90% local / 70% CI) cannot be
verified in this sandbox for the same reason** — `make coverage` depends on
`go test` succeeding across the full package set.

### Task 8.3 — Final deny-list non-regression check (full 7-PR chain)

```
git diff b14c726 HEAD -- '*.go' | rg '^[+-].*dagger\.io/dagger'   → empty (exit 1)
git diff b14c726 HEAD -- '*.go' | rg '^[+-].*go-kit-logger'        → empty (exit 1)
```

Confirmed empty across the **entire 7-commit chain** (`b14c726` baseline
through this PR's commit `7160e46`), not just this work unit — the final,
most important non-regression gate for the two most critical deny-listed
dependencies. Both the `dagger.io/dagger` SDK import/usage and the
`go-kit-logger` private-module import/usage are byte-identical to baseline
across all 7 PRs.

### Files Changed (this work unit)

| File | Action | What Was Done |
|------|--------|---------------|
| `Makefile` | Modified | Header comment `# Syntegrity Dagger Makefile` → `# Shipwright Makefile`; help-banner echo `🚀 Syntegrity Dagger - Makefile` → `🚀 Shipwright - Makefile`. |
| `main_test.go` | Modified | Reworded a comment referencing "the legacy Syntegrity Dagger identity" to avoid the residual token while preserving intent; the `NotContains("syntegrity"/"Syntegrity")` assertions were correctly left untouched. |
| `openspec/specs/README.md` | Modified (untracked, not committed) | `syntegrity-dagger` → `shipwright` in the directory-purpose sentence. Consistent with PR6 precedent: `openspec/` has never been committed on this chain. |

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `rg` residual sweeps (7.1, 7.2) are the harness for this unit's own scope — both confirmed clean/fully-classified as documented above. `gofmt -l main_test.go` → clean. |
| Runtime harness command/scenario and exact result | `go build ./...` and `go test -race ./...` — both **environment-blocked** (identical pre-existing `go-kit-logger` credential wall, confirmed byte-identical against unmodified baseline `b14c726` via a dedicated worktree). Not a regression. Real verification deferred to the user/CI. |
| Rollback boundary | Revert the 2 git-tracked files on this branch (`Makefile`, `main_test.go`; 3 insertions / 3 deletions) — independent of PR1-6; reverting PR7 alone restores the residual Makefile/comment strings without touching module path, binary name, config discovery, CI/CD, docs, or comments/fixtures from earlier work units. The untracked `openspec/specs/README.md` edit is outside git history entirely (same as PR6's `openspec/config.yaml` precedent). |

### Deviations from Design

None. The 4 residual hits found in 7.1 were exactly the kind of catch-all
cleanup this phase exists for — not itemized individually in tasks.md
because they were only discoverable by running the literal sweep command,
consistent with how Phase 6's `internal/app/app.go`/`health.go` hits were
discovered by its own `rg` sweep rather than pre-listed.

### Issues Found

None new. Confirms the `.serena/project.yml` residual `project_name:
"syntegrity-dagger"` flagged in PR6's Issues Found remains correctly
out of scope: `git ls-files .serena/` returns empty (genuinely untracked,
not part of git history on any branch of this chain) and it was excluded
from both 7.1/7.2 sweep commands via `--glob '!.serena/**'` per the task
instruction. No further action taken.

### Known environment limitation (final confirmation, unchanged from PR1-6)

The `go-kit-logger` sandbox credential wall documented since PR1 blocks
`go build ./...`, `go test -race ./...`, and `make coverage` in this
sandbox. This work unit performed the most rigorous confirmation yet that
this is genuinely pre-existing and not a rename artifact: a dedicated `git
worktree` against the **unmodified baseline commit `b14c726`** reproduces
the exact same error byte-for-byte. **This blocks final merge-readiness
confirmation for all 7 PRs in this chain and MUST be run by the user or CI
with real GitHub credentials for `github.com/getsyntegrity/go-kit-logger`
before merging.**

## FINAL SUMMARY — Full 7-PR Chain (Shipwright Project Rebranding)

### Chain overview

| PR | Branch | Commit | Phase(s) | Scope |
|---|---|---|---|---|
| 1 | `feature/shipwright-rebrand-pr1-module-imports` | `6f123bd` | 0-1 | Module path + imports |
| 2 | `feature/shipwright-rebrand-pr2-cli-build` | `25d5539` | 2 | CLI/build identity |
| 3 | `feature/shipwright-rebrand-pr3-config-env` | `11b28bf` | 3 | Config & env identity |
| 4 | `feature/shipwright-rebrand-pr4-ci-release` | `44cda3f` | 4 | CI/CD & release |
| 5 | `feature/shipwright-rebrand-pr5-docs-examples` | `46ed0ab` | 5 | Docs & examples |
| 6 | `feature/shipwright-rebrand-pr6-comments-fixtures` | `a7bcaf6` | 6 | Comments/fixtures/manual edits |
| 7 | `feature/shipwright-rebrand-pr7-residual-sweep` | `7160e46` | 7-8 | Residual sweep + final verification |

All 7 branches are local, stacked (`stacked-to-main` chain strategy — each
PR targets the previous PR's branch), **not pushed, no PRs opened**. Tip of
the chain: `feature/shipwright-rebrand-pr7-residual-sweep` @ `7160e46`.

### Totals across the full chain (`b14c726` baseline → `7160e46` tip)

- **7 commits**, one per work unit, all Conventional Commits.
- **99 files changed**, **901 insertions / 877 deletions** (1778 changed
  lines total across the chain — well distributed across 7 PRs, each
  individually reviewable; PR4 (402 lines) and PR5 (1042 lines) were the
  only two that exceeded the 400-line single-PR guard, both explicitly
  flagged in their own apply-progress records as atomic, indivisible
  work-unit boundaries rather than silently absorbed).
- Module path renamed: `github.com/getsyntegrity/syntegrity-dagger` →
  `github.com/pablogore/shipwright`.
- Binary renamed: `syntegrity-dagger` → `shipwright`.
- Config filename renamed: `.syntegrity-dagger.yml` → `.shipwright.yml`
  (6-candidate discovery list).
- Env prefix renamed: `SYNTEGRITY_DAGGER_` → `SHIPWRIGHT_`.
- CI/CD composite action directory renamed:
  `.github/actions/syntegrity-dagger/` → `.github/actions/shipwright/`.
- GitHub secret renamed: `SYNTEGRITY_DAGGER_TOKEN` → `SHIPWRIGHT_TOKEN`
  (⚠️ **out-of-band owner action required**: the `SHIPWRIGHT_TOKEN`
  repository + Dependabot secret must exist on GitHub before PR4 merges,
  flagged in PR4's apply-progress record).
- 30 docs/examples files rewritten to the Shipwright identity (PR5).
- 4 residual comment/fixture/config files cleaned up (PR6).
- 4 final residual hits caught and fixed by the Phase 7 sweep (PR7:
  `main_test.go` comment, `openspec/specs/README.md`, 2× `Makefile` lines).

### What's verified vs. what needs real-credential verification

**Verified in this sandbox (structural/static, no network dependency):**
- Both residual-identity sweeps (`syntegrity[-_ ]?dagger` and bare
  `syntegrity`) are clean/fully classified as of the PR7 tip commit.
- `git diff` over `dagger.io/dagger` and `go-kit-logger` import/usage lines
  is empty across the entire 7-commit chain — zero regression on the two
  most critical deny-listed dependencies.
- `gofmt -l` clean on every file touched by this SDD change (pre-existing,
  unrelated baseline `gofmt` violations in `internal/cache/*`,
  `internal/config/validation*.go`, `internal/config/yaml_parser.go`,
  `internal/config/yaml_step_config*.go`,
  `internal/pipelines/go-service/pipeline_test.go`,
  `internal/pipelines/test/gotester.go` remain, untouched, out of scope).
- `yamllint` on all 5 changed YAML files (PR4) — zero new error/warning
  categories vs. baseline.
- 3 Go packages with no transitive `go-kit-logger` dependency
  (`internal/cache`, `internal/interfaces`, `internal/pipelines`) build and
  `go test -race` cleanly against the fully renamed module path.
- Every RED-first TDD task (Phase 2, Phase 3) has documented RED evidence
  via compile-time/structural reasoning (execution blocked, but the
  guarantee is unambiguous: e.g. an assertion on a constant that doesn't
  yet exist can only fail).

**MUST be run by the user or CI with real GitHub credentials before merging
any of these 7 PRs** (all blocked by the identical pre-existing
`github.com/getsyntegrity/go-kit-logger` credential wall, confirmed
byte-identical against the unmodified `b14c726` baseline in this work
unit's dedicated worktree comparison — not a rename regression):
- `go build ./...` (full repo, all packages).
- `go test -race ./...` (full repo, all packages) — currently 13/16
  testable packages cannot even reach the test-execution stage in this
  sandbox.
- `make coverage` / coverage threshold validation (≥ 90% local / 70% CI).
- `make build` and runtime verification of `./shipwright --help` /
  `--version` (CLI-facing identity, currently verified only by source
  inspection in PR2).
- Runtime verification of `SHIPWRIGHT_TOKEN` env var recognition and
  `.shipwright.yml` config discovery (currently verified only by source
  inspection in PR3).
- GitHub Actions workflow dry-run (`ci.yml`, `release.yml`,
  `dependabot.yml`) — cannot execute `act` or real GitHub Actions locally.
- Creation of the `SHIPWRIGHT_TOKEN` GitHub repository + Dependabot secret
  (out-of-band owner action, flagged in PR4).
- Actual GitHub repository rename from `pablogore/syntegrity-dagger` to
  `pablogore/shipwright` (out-of-band owner action per the spec's
  "Repository renamed as an operational step" scenario — not a code task).

### Residual-sweep classification table (final, PR7 tip)

| Sweep | Result | Classification |
|---|---|---|
| `syntegrity[-_ ]?dagger` (product pattern) | **0 hits** | Fully clean — zero exceptions needed; even the two originally-anticipated documented exceptions (Makefile `gitlab.com/syntegrity` grep, root `1export`) don't actually match this pattern (no "dagger" token), so the true clean state is zero, not two. |
| bare `syntegrity` (company pattern) | 106 hits / 30 files | 100% classified: ~86 lines deny-listed company/org identity (`go-kit-logger`, `SyntegrityInfraPipeline`/`syntegrity-infra`, ssh/https cloner company strings, Makefile coverage grep, `tenant-svc.yml` registry, `AGENTS.md` eventengine); 5 lines company-owned/org-scoped example values explicitly covered by the spec's preserved-category scenario (`REQUIRE_CI_SUCCESS.md` org API examples, `go-service` pipeline test fixture URL); 3 lines legitimate `NotContains` test literals proving old-identity absence, not residual references. **Zero unclassified hits.** |
| `openspec/changes/shipwright-project-rebranding/**` exclusion | Confirmed | Both sweep commands excluded this path via `--glob`; SDD artifacts intentionally retain the old name for historical documentation. |

### Next step

This is the final work unit of the 7-PR chain. All 8 phases of tasks.md are
now marked complete (Phase 8's build/test gates marked complete with an
explicit environment-blocked annotation, per the same convention used
throughout PR1-6, rather than a bare checkmark implying full runtime
verification). Recommend `sdd-verify` next, followed by pushing the 7-branch
stack and opening PRs 1→7 in dependency order once the user/CI confirms
`go build`/`go test`/`make coverage` pass with real credentials.
