# Design: Wire `spec.source.repo` / `spec.source.ref` Into Workflow Source Resolution

## Technical Approach

Replace the "not implemented" stub in `resolveWorkflowSource` (main.go:563-573) with a branching path: when `spec.Repo` is non-empty, detect the clone protocol from the URL prefix and delegate to `shared.CloneRepo`. The existing `spec.Path` code path remains unchanged. Add `ctx context.Context` as the first parameter — the single caller `runWorkflowEngine` already holds one.

## Architecture Decisions

### Decision: Default ref to "main" before calling CloneRepo

**Choice**: Default `spec.Ref` to `"main"` in `resolveWorkflowSource` before passing to `CloneRepo`.
**Alternatives considered**: Let `HTTPSCloner.Clone` handle the default (it already does at line 92-94).
**Rationale**: `validateOpts` (cloner.go:47) runs before `CloneRepo`'s `Clone` and rejects empty `Branch`. The HTTPS cloner's internal default is unreachable for empty-branch inputs. Defaulting upstream keeps the contract clean: callers always pass a valid branch.

### Decision: Plain string protocol, no constants

**Choice**: Use `"ssh"` as the protocol string, matching `CloneRepo`'s existing convention (cloner.go:39).
**Alternatives considered**: Define `SSHProtocol`/`HTTPSProtocol` constants in `shared`.
**Rationale**: Out-of-scope per proposal (no changes to `cloner.go`). The string literal `"ssh"` is the existing contract; adding constants would require modifying `cloner.go`.

### Decision: No changes to cloner internals

**Choice**: Call `shared.CloneRepo` as-is; no modifications to cloner packages.
**Alternatives considered**: Add retry/timeout configuration in the new call site.
**Rationale**: `CloneRepo` already handles retries (HTTPS: 3 attempts with 2s delay), timeout (5m), and credential resolution. Duplicating this logic would diverge from the shared cloner pattern.

## Data Flow

```
runWorkflowEngine(ctx, manifest, graph, flags, reg, client)
    │
    └─→ resolveWorkflowSource(ctx, client, spec)
            │
            ├─ spec.Repo == "" → client.Host().Directory(path)  [unchanged]
            │
            └─ spec.Repo != "" →
                    protocol = hasPrefix("git@") ? "ssh" : "https"
                    ref = spec.Ref == "" ? "main" : spec.Ref
                    opts = GitCloneOpts{Repo, Branch: ref, Name: "workflow-source"}
                    shared.CloneRepo(ctx, client, opts, protocol) → *dagger.Directory
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `main.go` | Modify | Add `ctx context.Context` param to `resolveWorkflowSource`; add `strings` import; implement protocol detection and clone delegation |
| `main_test.go` | Modify | Add tests for repo/ref code path (HTTPS, SSH, empty ref default, path fallback) |

## Interfaces / Contracts

```go
// BEFORE
func resolveWorkflowSource(client *dagger.Client, spec manifest.SourceSpec) (*dagger.Directory, error)

// AFTER
func resolveWorkflowSource(ctx context.Context, client *dagger.Client, spec manifest.SourceSpec) (*dagger.Directory, error)
```

Caller update at main.go:484:
```go
// BEFORE
source, err := resolveWorkflowSource(client, m.Spec.Source)

// AFTER
source, err := resolveWorkflowSource(ctx, client, m.Spec.Source)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Protocol detection from URL prefix | Table-driven: `git@github.com:org/repo.git` → SSH, `https://github.com/org/repo.git` → HTTPS, `http://github.com/org/repo.git` → HTTPS |
| Unit | Ref defaulting | Empty ref → "main", explicit ref preserved |
| Unit | Path fallback unchanged | `spec.Repo == ""` → `client.Host().Directory(path)` |
| Unit | CloneRepo delegation | Verify correct opts constructed (mock `CloneRepo` or use real Dagger client with invalid repo → expect error) |

Specific test cases for `main_test.go`:

1. **TestResolveWorkflowSource_HTTPSClone** — `spec.Repo = "https://github.com/org/repo.git"`, verify `CloneRepo` called with HTTPS protocol (error expected since no real repo, proves delegation path is reached)
2. **TestResolveWorkflowSource_SSHClone** — `spec.Repo = "git@github.com:org/repo.git"`, verify SSH protocol path (error expected, proves SSH detection)
3. **TestResolveWorkflowSource_EmptyRefDefaultsToMain** — `spec.Repo = "https://github.com/org/repo.git"`, `spec.Ref = ""`, verify error message includes "main" branch
4. **TestResolveWorkflowSource_ExplicitRefPreserved** — `spec.Repo = "https://github.com/org/repo.git"`, `spec.Ref = "develop"`, verify error includes "develop"
5. **TestResolveWorkflowSource_PathFallback** — `spec.Repo = ""`, `spec.Path = "./src"`, verify no clone, returns `client.Host().Directory("./src")`

Note: Tests 1-4 will use a real `dagger.Client` (matching existing test patterns in `https_cloner_test.go`) and expect clone failures for invalid repos. This proves the code path is exercised without needing live network access.

## Threat Matrix

| Boundary | Applicability | Design Response |
|----------|--------------|-----------------|
| Documentation-like paths | N/A — no executable file classification | — |
| Git repository selection | Applicable — URL prefix determines SSH vs HTTPS protocol | Protocol detection uses simple `strings.HasPrefix` on user-provided URL; clone runs inside Dagger container with no host access; `CloneRepo` validates inputs |
| Commit state | N/A — this change clones, never commits | — |
| Push state | N/A — no push operations | — |
| PR commands | N/A — no PR automation | — |

Expected safe behavior for Git repository selection: `git@` prefix → SSH clone (requires `SSH_PRIVATE_KEY` env or `~/.ssh/syntegrity` key), any other prefix → HTTPS clone (resolves credentials via `ResolveGitCredentials`). Failure behavior: missing SSH key returns cloner error; missing HTTPS credentials returns anonymous access (public repos only) or credential resolution error.

## Migration / Rollout

No migration required. The change is a behavioral replacement of a stub — reverting restores the "not implemented" error. No schema changes, no persisted state, no feature flags.

## Open Questions

- [ ] None — all design decisions resolved from proposal and specs.
