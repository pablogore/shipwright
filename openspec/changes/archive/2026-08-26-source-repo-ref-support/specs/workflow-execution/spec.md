# Delta for Workflow Execution

## MODIFIED Requirements

### Requirement: Manifest-Driven Entrypoint Replaces The Preset CLI Path In The Same Change

The manifest-driven execution entrypoint MUST land no later than the
removal of the `go-service` preset (see `composition-model`'s "No Named
Capability-Set Preset Ships"). The repository MUST NOT reach a merged state
where the CLI has neither the preset flag nor a working manifest-driven
entrypoint.

(Previously: only `spec.source.path` was supported by `resolveWorkflowSource`)

#### Scenario: CLI always has a working execution path

- GIVEN the repository state after the `go-service` preset and its CLI
  flag are removed
- WHEN the CLI is invoked with a workflow manifest
- THEN the manifest-driven entrypoint executes it successfully — there is
  no point in the merged history where the CLI can run neither path

#### Scenario: Git-based source resolves via clone

- GIVEN a manifest with `spec.source.repo: "https://github.com/org/repo.git"` and `spec.source.ref: "main"`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` clones the repository and returns a valid
  `*dagger.Directory` — the "not implemented" error is no longer raised

#### Scenario: SSH-based source resolves via clone

- GIVEN a manifest with `spec.source.repo: "git@github.com:org/repo.git"` and `spec.source.ref: "main"`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` clones via SSH protocol and returns a
  valid `*dagger.Directory`

#### Scenario: Missing ref with repo fails closed

- GIVEN a manifest with `spec.source.repo: "https://github.com/org/repo.git"` but no `spec.source.ref`
- WHEN the CLI is invoked with the manifest
- THEN `resolveWorkflowSource` returns an error — source.ref is required when source.repo is set
