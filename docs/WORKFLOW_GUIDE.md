# Authoring a Shipwright Workflow Manifest

A workflow manifest is a YAML document describing a DAG of steps — build,
test, scan, publish, deploy — that Shipwright's `--workflow` flag consumes
as its sole entrypoint. Every manifest is fully parsed and validated
(structure, references, and the dependency graph) before any container
starts, so most authoring mistakes fail fast with a specific error rather
than partway through a run.

## Quick path

1. **Write a manifest.** This repo has no `.shipwright/workflow.yaml`, so
   start from [`examples/workflow/minimal.yaml`](../examples/workflow/minimal.yaml)
   — the smallest manifest that actually runs, inlined below:

   ```yaml
   apiVersion: shipwright.dev/v1
   kind: Workflow
   metadata:
     name: minimal-go-service
   spec:
     source:
       path: .
     steps:
       - id: build
         capability: build
         uses:
           provider: go
           version: "1"
         with:
           goVersion: "1.26.1"

       - id: unit
         capability: test
         uses:
           provider: go-test
           version: "1"
         needs: [build]
         input: ${{ steps.build.output }}
   ```

2. **List its steps** — this only parses and validates, it never runs a
   container:

   ```sh
   shipwright --workflow examples/workflow/minimal.yaml --list-steps
   ```

3. **Run it.** Because this repo's `.shipwright/workflow.yaml` does not
   exist, pass `--workflow` explicitly every time:

   ```sh
   shipwright --workflow examples/workflow/minimal.yaml
   ```

For a richer, real-world shape — a `build` step fanning out into two
parallel test steps that fan back into one `publish` step — see
[`examples/workflow/diamond.yaml`](../examples/workflow/diamond.yaml).
Sections 5 and 8 below excerpt its `publish` step to illustrate wiring and
`when`.

## Manifest anatomy

A manifest has exactly four top-level fields:

| Field | Type | Notes |
|---|---|---|
| `apiVersion` | string | Only `shipwright.dev/v1` is accepted today. Any other value is rejected. |
| `kind` | string | Only `Workflow` is accepted today. |
| `metadata` | object | `name` (required), `description`, `labels`. |
| `spec` | object | Everything else — see below. |

`spec` has exactly seven fields:

| Field | Type | Purpose |
|---|---|---|
| `source` | object | The workflow's default input: `path` (local) or `repo`/`ref`/`authSecretRef` (Git). |
| `variables` | map of string→string | Named string values, referenced via `${{ variables.<name> }}`. |
| `secrets` | map of string→object | Named secret references (`fromEnv` only — see [Values](#values-variables-secrets-and-with)). |
| `steps` | list of Step | The DAG's nodes — see [Steps](#steps). |
| `execution` | object | Scheduling controls — see [Execution controls](#execution-controls). |
| `environments` | map | Deployment-target metadata — see [Declared but not enforced](#declared-but-not-enforced). |
| `policies` | object | Declared intent for a later enforcement layer — see [Declared but not enforced](#declared-but-not-enforced). |

An unrecognized field anywhere in the document is a **hard decode error**,
not a silently ignored typo — the parser rejects unknown fields before any
other validation runs. A manifest is also capped at 1 MiB; anything larger
is rejected unread.

## Steps

Every step has exactly eight fields:

| Field | Required | Example |
|---|---|---|
| `id` | yes, unique | `id: build` |
| `capability` | yes, one of `build`, `test`, `artifact`, `deploy`, `run` | `capability: build` |
| `uses` | yes | `uses: {provider: go, version: "1"}` |
| `needs` | no | `needs: [build]` |
| `input` | no | `input: ${{ steps.build.output }}` |
| `with` | no | `with: {goVersion: "1.26.1"}` |
| `when` | no | `when: {branch: [main]}` |
| `attempts` | no | `attempts: 3` |

`uses` selects **exactly one** of `provider` (an in-repo capability
implementation) or `module` (an external one) — never both meaningfully at
once — and both require a non-empty `version`:

```yaml
uses:
  provider: go-test   # or: module: <external-module-ref>
  version: "1"
```

`capability` is the contract; `uses` is the implementation. A step
declaring `capability: test` with `uses.provider: go-test` means "run
something that satisfies the Tester contract, specifically the `go-test`
provider" — the capability name and the provider name are independent.

## Wiring steps together

A step's position in `spec.steps` never implies an execution-order edge.
The only way to declare "run after" is `needs: [<step-id>, ...]`.

**`input`** resolves in one of two ways:

- **Unset** — falls back to `spec.source`, the workflow's default input
  directory.
- **Set** — must be *exactly one* `${{ steps.<id>.output }}` reference.
  Any other shape (a literal string, a `variables.`/`secrets.` reference,
  or more than one token) is rejected.

Reading `steps.<id>.output` from **any** field — `input` or `with` — also
requires `<id>` to appear in the reading step's own `needs: [...]`. If it
doesn't, the manifest is rejected while building the dependency graph, with an
`UndeclaredDataReferenceError`, before any step runs. This is why
`examples/workflow/diamond.yaml`'s `publish` step declares
`needs: [build, unit, vuln]` — its `input` reads `steps.build.output`, so
`build` must be present in `needs` even though `unit` and `vuln` don't
themselves feed `publish`'s `input`.

### The output-kind matrix

The capability a step declares fixes the *type* of its output, and each
output type is legal in exactly one downstream field:

| capability | produces | usable as downstream `input:` | readable in downstream `with:` |
|---|---|---|---|
| `build` | Directory | yes | no |
| `test` | File | no | no — unreachable today |
| `artifact` | string | no | yes |
| `deploy` | string | no | yes (no provider ships) |
| `run` | Container | no | no (no provider ships) |

The practical consequence: **a `test` step's output cannot be consumed by
any field.** A test step only ever appears in a downstream step's
`needs: [...]` for ordering. This is exactly why
`examples/workflow/diamond.yaml`'s `publish` step reads
`${{ steps.build.output }}` and not one of the test steps' outputs — there
is no field that could read a test step's output even if you wanted it to.

## Values: variables, secrets, and with

Interpolation supports **exactly three** namespaces inside `${{ ... }}` —
there is no fourth, and no expression syntax beyond a bare reference:

| Namespace | Syntax | Resolves to |
|---|---|---|
| Variables | `${{ variables.<name> }}` | The string value of `spec.variables.<name>`. |
| Secrets | `${{ secrets.<name> }}` | A typed secret handle for `spec.secrets.<name>` — never plaintext. |
| Step output | `${{ steps.<id>.output }}` | The output of a step declared in `needs: [...]` (see the output-kind matrix above). |

`variables` are strings only — there is no numeric, boolean, or nested
variable type.

A secret reference must be a field's **entire value**. You cannot
concatenate a secret with other text:

```yaml
with:
  creds: ${{ secrets.registry }}          # OK — the whole field is the secret
  creds: "user:${{ secrets.registry }}"   # Rejected — secret mixed with literal text
```

`secrets.<name>` supports only `fromEnv` — the name of an environment
variable holding the secret's value:

```yaml
secrets:
  registry:
    fromEnv: REGISTRY_PASSWORD
```

There is no `value:` field for embedding a literal secret inline. Writing
`secrets.registry.value: "..."` is not a validation error you discover
later — it fails immediately at decode, because the schema has no field to
hold it.

## Provider catalog

These are the only providers registered today, all pinned to `version: "1"`:

| Provider | Capability | `with` fields |
|---|---|---|
| `go` | `build` | `goVersion` (string), `binaryName` (string) |
| `go-test` | `test` | `coverage` (number) |
| `golangci-lint` | `test` | *(none)* |
| `govulncheck` | `test` | *(none)* |
| `container` | `artifact` | `ref` (string), `creds` (secret), `registryUser` (string) |

No `deploy` or `run` capability provider ships yet. A step declaring
`capability: deploy` or `capability: run` will fail to resolve at run
time — there is no provider to select.

## Conditional steps: when

`when` supports **exactly one** key today: `branch`.

| Key | Populated at runtime? | Evidence |
|---|---|---|
| `branch` | Yes — always | `main.go:498`, `Predicates: map[string]string{"branch": flags.branch}` |

Every other key you might write into `when` — anything other than
`branch` — is silently never matched: `matchesWhen`
(`internal/workflow/engine/execute.go:386`) returns `false` for any `when`
key absent from the predicates map, and `branch` is the only key that map
ever contains.

`--branch` defaults to `develop`. This means
`examples/workflow/diamond.yaml`'s `publish` step:

```yaml
when:
  branch: [main]
```

is **silently skipped** unless you run with `--branch main` explicitly:

```sh
# publish is skipped — --branch defaults to "develop"
shipwright --workflow examples/workflow/diamond.yaml

# publish runs
shipwright --workflow examples/workflow/diamond.yaml --branch main
```

A skipped step is reported with status `skipped`, not `failed` — the
workflow still exits successfully.

Values within one key are OR'd (`branch: [main, release]` matches either);
multiple keys are AND'd. Since `branch` is the only key ever populated,
adding any second key to `when` unconditionally skips the step, because
that second key can never be satisfied.

> **Not implemented today.** `when`'s schema type is a generic
> `map[string][]string`, which could in principle carry more predicates in
> a future release. No such predicate exists yet, and none is documented
> here as usable syntax.

## Execution controls

`attempts` is the **total** number of attempts a step is allowed, not the
number of *additional* retries after a first failure:

```yaml
attempts: 3   # runs at most 3 times total, not 1 + 3
```

A value below `1` (including `0`) normalizes to `1` — a step always runs
at least once. When set, a step's own `attempts` overrides the
workflow-level retry default; when unset, the step falls back to that
default.

There is **no workflow-level `spec.execution` field for retries at all** —
`spec.execution` only carries `concurrency`, `failFast`, and `timeout`.
Per-step `attempts` is the only way to configure retry behavior.

`timeout` is a single per-step duration string (parsed with Go's
`time.ParseDuration`, e.g. `30m`, `90s`) applied as each step's own
deadline — it is not a total-workflow timeout.

## Declared but not enforced

The following fields parse and validate successfully but change no
runtime behavior in the current engine. They are documented explicitly,
rather than omitted, because `examples/workflow/diamond.yaml` itself uses
them — a reader who copies that manifest should not assume they do
something they don't.

| Field | Parses/validates | Actually does |
|---|---|---|
| `spec.execution.concurrency.maxParallel` | Must be `>= 0`. | Recorded only. Execution is strictly sequential, wave by wave and step by step within a wave — `maxParallel` never widens it. |
| `spec.environments.<name>.approvals` | Free-form `required: [...]` list. | Nothing. It is queryable metadata; no layer blocks on it. |
| `spec.policies.*` | Parses into typed fields (`secrets`, `providers`, `dependencies`, `artifacts`). | Mostly nothing. The one exception: an empty `uses.version` is *always* rejected, independent of `policies.providers.requireVersion`'s value. |

## CLI reference

| Flag | Default | Purpose |
|---|---|---|
| `--workflow` | `.shipwright/workflow.yaml` | Path to the manifest to run. This repo has no file at the default path — pass it explicitly. |
| `--list-steps` | `false` | Parse, validate, and print the manifest's steps without running anything. |
| `--step` | *(empty — run everything)* | Run one step and everything it transitively `needs`. |
| `--branch` | `develop` | The value bound to `when: {branch: [...]}`'s only predicate. |

## Gotchas

| Trap | Reality |
|---|---|
| `when: {branch: [main]}` with default `--branch develop` | Step silently skips; the workflow still succeeds. |
| `attempts: 3` | 3 total attempts, not 1 initial + 3 retries. `attempts: 0` still means 1. |
| Looking for a workflow-level retry setting | Doesn't exist. `spec.execution` has no retries field at all. |
| `maxParallel: 4` | Recorded, never honored — execution is sequential. |
| `creds: "user:${{ secrets.tok }}"` | Hard error — a secret must be a field's entire value, never mixed with other text. |
| `secrets.x.value: "..."` | Decode error — only `fromEnv` exists; there is no way to embed a literal secret. |
| Reading `steps.X.output` without `X` in `needs: [...]` | `UndeclaredDataReferenceError` at graph-build time, before any step runs. |
| Trying to consume a `test` step's output | Impossible — a `test` step's File output has no consuming field. |
| Running with just `--workflow` (no value) in this repo | Fails — `.shipwright/workflow.yaml` doesn't exist here. Pass `--workflow examples/workflow/minimal.yaml` (or `diamond.yaml`). |
| `approvals: {required: [...]}` | Parsed and stored; never gates anything. |

## Error reference

| You see | Cause | Fix |
|---|---|---|
| `manifest: decode: ...` (mentioning an unrecognized field name) | The manifest contains a field the schema doesn't define — decoding rejects unknown fields outright. | Remove or rename the offending field; check spelling and nesting against the tables above. |
| `UndeclaredDataReferenceError` | A field reads `${{ steps.<id>.output }}` but `<id>` is missing from that step's own `needs: [...]`. | Add `<id>` to `needs: [...]`. |
| `interp: cannot concatenate secret reference ... with other content — a secret must be a field's entire value, never mixed with literal text or other references` | A `with` field mixes a secret reference with literal text or another reference. | Make the field's entire value the single secret reference, or move the secret to its own field. |

## Next steps

- [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) — the engine's internal
  execution pipeline, if you need to understand *how* a step runs rather
  than *what to write*.
- [`docs/API.md`](API.md) — the `pkg/shipwright` capability interfaces a
  provider implements.
- [`../COMPATIBILITY.md`](../COMPATIBILITY.md) — public-surface
  compatibility guarantees.
