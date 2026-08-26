# Git Source Resolution Specification

## Purpose

Resolves a workflow's input source from a remote git repository via HTTPS or
SSH clone. Delegates to `internal/pipelines/shared.CloneRepo` using the
protocol detected from the repository URL prefix.

## Requirements

### Requirement: Protocol Detection From Repository URL

The system MUST detect the clone protocol from the `spec.source.repo` URL
prefix. URLs starting with `git@` or `ssh://` MUST use SSH; all other
URLs MUST use HTTPS.

#### Scenario: SSH URL detected for git@ prefix

- GIVEN `spec.source.repo` is `git@github.com:org/repo.git`
- WHEN protocol detection runs
- THEN the system selects SSH protocol

#### Scenario: SSH URL detected for ssh:// scheme

- GIVEN `spec.source.repo` is `ssh://git@github.com/org/repo.git`
- WHEN protocol detection runs
- THEN the system selects SSH protocol

#### Scenario: HTTPS URL detected for https:// prefix

- GIVEN `spec.source.repo` is `https://github.com/org/repo.git`
- WHEN protocol detection runs
- THEN the system selects HTTPS protocol

### Requirement: Explicit Ref Required When Repo Is Set

When `spec.source.repo` is non-empty, `spec.source.ref` MUST be
non-empty. An omitted or empty ref is a manifest error — the system
MUST NOT silently default to a branch name. This enforces pinned
source semantics for reproducible CI/CD.

#### Scenario: Empty ref with repo fails closed

- GIVEN `spec.source.repo` is `https://github.com/org/repo.git` and `spec.source.ref` is `""`
- WHEN the system resolves the workflow source
- THEN it returns an error containing "source.ref is required"

#### Scenario: Explicit ref preserved

- GIVEN `spec.source.ref` is `develop`
- WHEN the system resolves the workflow source
- THEN the clone uses branch `"develop"`

### Requirement: Clone Delegates To shared.CloneRepo

When `spec.source.repo` is non-empty, the system MUST call
`shared.CloneRepo(ctx, client, GitCloneOpts{...}, protocol)` and return
its result. The `GitCloneOpts.Name` field MUST be `"workflow-source"`.

#### Scenario: HTTPS clone succeeds

- GIVEN `spec.source.repo` is a valid HTTPS URL and `spec.source.ref` is set
- WHEN `resolveWorkflowSource` is called
- THEN `shared.CloneRepo` is invoked with HTTPS protocol and a valid `*dagger.Directory` is returned

#### Scenario: SSH clone succeeds

- GIVEN `spec.source.repo` is a valid SSH URL (`git@...` or `ssh://...`) and `spec.source.ref` is set
- WHEN `resolveWorkflowSource` is called
- THEN `shared.CloneRepo` is invoked with SSH protocol and a valid `*dagger.Directory` is returned

### Requirement: AuthSecretRef Fails Closed

When `spec.source.authSecretRef` is non-empty, `resolveWorkflowSource`
MUST return an explicit error. The field exists in the schema but is
not yet wired — silently ignoring it would be misleading.

#### Scenario: Non-empty authSecretRef rejected

- GIVEN `spec.source.repo` is set and `spec.source.authSecretRef` is `"github-prod"`
- WHEN `resolveWorkflowSource` is called
- THEN it returns an error containing "authSecretRef is not supported yet"

### Requirement: Context Propagation

`resolveWorkflowSource` MUST accept `ctx context.Context` as its first
parameter. The caller already holds a context and MUST pass it through.

#### Scenario: Context passed to CloneRepo

- GIVEN the caller invokes `resolveWorkflowSource(ctx, client, spec)`
- WHEN `spec.Repo` is non-empty
- THEN the context is forwarded to `shared.CloneRepo`

### Requirement: Path Fallback Unchanged

When `spec.source.repo` is empty, the system MUST fall through to the
existing path-based local directory resolution with no behavioral change.

#### Scenario: Path-based resolution unchanged

- GIVEN `spec.source.repo` is `""` and `spec.source.path` is `"./src"`
- WHEN `resolveWorkflowSource` is called
- THEN the system returns `client.Host().Directory("./src")` with no clone

### Requirement: Clone Failure Propagates Error

If `shared.CloneRepo` returns an error, `resolveWorkflowSource` MUST
propagate it without modification. The error MUST include the upstream
message (e.g. missing SSH key, invalid ref, network failure).

#### Scenario: SSH key missing returns cloner error

- GIVEN `spec.source.repo` is an SSH URL and no SSH key is available
- WHEN `shared.CloneRepo` fails
- THEN `resolveWorkflowSource` returns the cloner's error unchanged

#### Scenario: Invalid ref returns cloner error

- GIVEN `spec.source.ref` is `nonexistent-branch`
- WHEN the clone attempts to check out the ref
- THEN `resolveWorkflowSource` returns the cloner's error unchanged
