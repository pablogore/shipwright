# Proposal: Wire `spec.source.repo` / `spec.source.ref` Into Workflow Source Resolution

## Intent

`spec.source.repo` and `spec.source.ref` parse successfully but fail at runtime with an explicit "not implemented" error in `resolveWorkflowSource`. The schema already defines these fields (`SourceSpec` in `internal/workflow/manifest/schema.go:50`), and the git clone logic already exists in `internal/pipelines/shared`. This change connects the two: when `spec.Repo != ""`, `resolveWorkflowSource` clones the repository and returns the cloned directory instead of returning an error.

## Scope

### In Scope

- Rewrite `resolveWorkflowSource` to handle `spec.Repo` (git clone) and `spec.Path` (existing behavior)
- Detect protocol (`git@` → SSH, else HTTPS) from the URL prefix
- Delegate to `shared.CloneRepo(ctx, client, GitCloneOpts{...}, protocol)`
- Add `ctx context.Context` parameter to `resolveWorkflowSource` (callers already have it)
- Tests for both code paths: `repo` set (HTTPS and SSH) and `path` set (existing)
- Default ref to `"main"` when `spec.Ref` is empty

### Out of Scope

- `authSecretRef` wiring — the schema field exists but has no runtime resolver; document as follow-up
- Shallow clone support for SHA-based refs (requires `--depth` flag on the cloner)
- Changes to `internal/pipelines/shared/cloner.go` or `internal/pipelines/shared/https_cloner.go`

## Capabilities

### New Capabilities

- `git-source-resolution`: Resolves workflow source from a remote git repository via HTTPS or SSH clone, returning a `*dagger.Directory` to the engine

### Modified Capabilities

- `workflow-execution`: The `resolveWorkflowSource` function in `main.go` gains a new code path; execution behavior unchanged for `spec.source.path`

## Approach

1. Add `ctx context.Context` as first parameter to `resolveWorkflowSource` — the single caller (`runWorkflowEngine` at line 484) already holds a context
2. When `spec.Repo != ""`:
   - Determine protocol: `strings.HasPrefix(spec.Repo, "git@")` → `shared.SSHProtocol`, else `shared.HTTPSProtocol`
   - Build `shared.GitCloneOpts{Repo: spec.Repo, Branch: spec.Ref, Name: "workflow-source"}`
   - Call `shared.CloneRepo(ctx, client, opts, protocol)` and return the result
3. When `spec.Repo == ""`: existing path logic (unchanged)
4. Add tests: mock `dagger.Client` with `pipelines.DaggerClient` interface; test HTTPS clone, SSH clone, empty ref default, and path fallback

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `main.go:563-573` | Modified | `resolveWorkflowSource` rewritten to support `spec.Repo` |
| `main_test.go` | Modified | New tests for repo/ref code path |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Empty `ref` defaults to `"main"` for HTTPS but SSH clone may fail without key | Medium | Default ref to `"main"`; SSH auth failure returns clear error from existing cloner |
| Shallow clone (`--depth=1`) incompatible with SHA-based refs | Low | Not addressed in this change; shallow clone is a shared concern, not this function's |
| `authSecretRef` parses but has no runtime effect | Low | Explicitly out of scope; documented as follow-up |
| Breaking existing `spec.source.path` behavior | Low | Path code path is unchanged; guarded by `spec.Repo == ""` |

## Rollback Plan

Revert the commit. The only behavioral change is in `resolveWorkflowSource`; reverting restores the "not implemented" error for `spec.Repo`. No data migration, no schema changes, no persisted state affected.

## Dependencies

- `internal/pipelines/shared/cloner.go` (`CloneRepo` factory)
- `internal/pipelines/shared/https_cloner.go` (HTTPS auth)
- `internal/pipelines/shared/ssh_cloner.go` (SSH auth)
- `internal/pipelines/shared/credentials.go` (token resolution)

## Success Criteria

- [ ] `spec.source.repo` with an HTTPS URL clones the repository and returns a valid `*dagger.Directory`
- [ ] `spec.source.repo` with an SSH URL (`git@...`) clones via SSH
- [ ] Empty `spec.source.ref` defaults to `"main"` branch
- [ ] Existing `spec.source.path` behavior unchanged
- [ ] Unit tests pass for both code paths (`go test -race ./...`)
