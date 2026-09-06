# Release Integrity Contract (RELEASE-DIST-01A)

This document defines the invariants a Shipwright root release (`v*` tag)
must satisfy, and how they're enforced in `.github/workflows/release.yml`.
It does not cover Dagger-based build/publish (RELEASE-DIST-01B) or consumer
bootstrap/install tooling (RELEASE-DIST-01C) — see the roadmap for those.

## Release identity

A modern Shipwright release is identified by the triple **Tag ↔ Commit ↔
GitHub Release**, plus a corresponding `CHANGELOG.md` entry:

- The tag is the source of truth for the version. It matches
  `^v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$` (enforced
  by the "Validate release version shape" step; extracted and tested by
  `internal/releaseguard`'s `TestRootReleaseWorkflow_ShapeRegex`). Major
  version is capped at `0|1` because the root module has no `/vN` path
  suffix yet — a major bump requires that suffix first (Go's module
  major-version rule), not just a bigger number in the tag.
- The tag is **immutable** once pushed: `scripts/release-tag-guard.sh`
  creates it if absent, treats "already exists, same commit" as a safe
  retry, and **fails the job** if it exists pointing at a different commit.
  It never deletes, moves, or force-pushes a tag.
- This guard runs for **every** release path -- a tag push, a CI-triggered
  automatic release, and a genuinely manual admin `workflow_dispatch` alike.
  There is no trigger-dependent second semantics where a manual release
  gets a weaker guarantee: if it produces a release, it produces a tag
  under the same fail-closed rule. For a tag push the tag already exists at
  the checked-out commit, so the guard is a same-commit no-op there; for an
  admin dispatch with no pre-existing tag, the guard is what creates it.
- The GitHub Release is created by GoReleaser against that exact tag/commit
  (`scripts/verify-release-asset.sh` re-downloads a real asset post-publish
  and executes it to confirm the reported version and commit match).
- `CHANGELOG.md` gets a `## Release <tag>` entry via
  `scripts/changelog-prepend-release.sh`, prepended above prior history.

## Distribution identity

- **Current contract**: assets named `shipwright-<os>-<arch>` (raw
  binaries) and `shipwright_<version>_<Os>_<Arch>.tar.gz`/`.zip`
  (archives), plus a combined `checksums.txt` covering both. Root tag
  namespace is `v*`.
- **Provider contract**: `providers/go/vX.Y.Z` and `providers/rust/vX.Y.Z`
  are separate, disjoint tag/release lines with their own workflows and
  their own SemVer validation. A provider tag can never satisfy the root
  release's shape regex (it requires a leading `v`, not `providers/...`),
  and root's `git describe --tags --match 'v[0-9]*'` calls can never
  resolve a provider tag as the root's "latest" — both guarded by
  `internal/releaseguard`.

## Historical / legacy releases

`v0.4.0-beta.1`, `v0.4.1`, and `v0.5.0` predate this contract: they were
published under the pre-rebrand `syntegrity-dagger-*` asset naming, with no
corresponding Git tag resolvable in current history (GitHub only recorded
`target_commitish: develop`, a branch name, not a commit). They are marked
**legacy** in `CHANGELOG.md` and in their GitHub Release descriptions.
They are not consumable under the current contract and are not treated as
verified — no tag was recreated and no asset was altered to make them look
retroactively compliant. The first `vX.Y.Z` release produced under this
contract is the actual point-zero of the modern distribution line.

## Retry semantics

- **Same commit, same tag, re-run** (e.g. a transient failure after the tag
  was already pushed): safe no-op on the tag step; the CHANGELOG step is
  idempotent by tag (`changelog-prepend-release.sh` skips if `## Release
  <tag>` is already present) — a re-run does not duplicate the entry.
- **Tag exists at a different commit**: fail closed, exit 1, no mutation.
  This must be resolved by a human, never by the workflow.
- **Any real git failure** (commit or push) during the CHANGELOG update now
  fails the job — the previous `|| echo "..."` fallbacks that swallowed
  real errors were removed. The only silent case is the correct one:
  nothing staged to commit.
