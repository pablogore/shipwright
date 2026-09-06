---
name: shipwright-security
description: "Security-sensitive changes to secrets handling, registry auth, plugin loading, or provider/plugin YAML. Trigger: secrets, credentials, registry, plugin, external input."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## When to Use

Load this skill when a change touches:

- Secrets or credentials (Git tokens, registry passwords, SSH keys) — `internal/pipelines/shared/credentials.go`, `internal/pipelines/shared/docker.go`.
- Container registry authentication — anything calling `Container.WithRegistryAuth` or `dagger.Client.SetSecret`.
- Plugin loading — `internal/plugins/loader.go`, `internal/plugins/registry.go`.
- Parsing of provider/plugin/pipeline YAML that ultimately reaches command execution or plugin config — `internal/config/yaml_parser.go`, `internal/config/yaml_step_config.go`, plugin `Initialize()` methods reading `PluginContext.GetConfiguration()`.
- Any code that builds a shell string or `exec.Command`/`dagger.Container.WithExec` argument list from config- or YAML-sourced values.

This skill was authored from scratch for Shipwright's actual attack surface (there is no upstream security skill to port); the rules below are grounded in the code as it exists in this repo, not generic advice.

## Attack Surface In This Repo

Shipwright's untrusted-input boundary is **provider/plugin YAML and environment variables**, not user HTTP input — this is a CLI that wraps Dagger for CI/CD execution, so "untrusted" here means "content a pipeline author or PR contributor controls that this process will parse, then act on with real credentials." Three concrete surfaces exist today:

1. **Registry/Git credentials.** `internal/pipelines/shared/credentials.go` resolves Git auth from `GITHUB_TOKEN`, `CI_JOB_TOKEN`, `GITLAB_PAT`, `SSH_PRIVATE_KEY` (env-var cascade). `internal/pipelines/shared/docker.go` wraps the registry password in `client.SetSecret("registry-pass", pass)` before handing it to `WithRegistryAuth`.
2. **Plugin loading.** `internal/plugins/loader.go` (`LoadFromFile`/`LoadFromConfig`) can `plugin.Open()` an arbitrary `.so` file whose path comes from plugin config (`type: file`, `path: ...`), which can itself originate from parsed pipeline YAML.
3. **Command construction from config values.** Several code paths build a `[]string` argv or a shell string from config-sourced strings (Nomad job name/content, image name/tag, registry URL) before passing it to `exec.Command` or `dagger.Container.WithExec`.

## Hard Rules

### Secrets

- **Never pass a credential as a plain Go `string` into `WithExec`, `WithEnvVariable`, or `fmt.Sprintf` into a command.** Wrap it with `client.SetSecret(name, value)` first and pass the resulting `*dagger.Secret`, exactly as `internal/pipelines/shared/docker.go` does for the registry password. A secret baked into `WithExec`/`WithEnvVariable` as a plain string is not scrubbed from Dagger's build cache or container history.
- **Never log a secret's raw value.** `GitCredentials.String()` in `internal/pipelines/shared/credentials.go` is the pattern to follow: it logs `Source`, `User`, and `ExpiresAt`, and deliberately omits `Token`. Any new credential-carrying struct should implement `String()`/`LogValue()` the same way, and any `logger.L()` call touching credentials should be reviewed for accidental field inclusion (e.g. `"config", p.config` style logging — check the map doesn't contain a raw secret before logging it wholesale).
- Treat `ValidateRequiredSecrets` (same file) as the pattern for gating operations that need auth (`push`, `tag`, `release`) — extend that map rather than adding ad hoc auth checks elsewhere.
- Anonymous/no-credential fallback exists for read-only Git operations only; never let it silently satisfy an operation that requires auth.

### Container registries

- Registry auth always goes through `WithRegistryAuth(imageRef, user, *dagger.Secret)` — never construct a `docker login`-style shell command.
- Registry URLs must pass `config.ValidateRegistryURL` (in `internal/config/validation.go`) before use; it enforces `http`/`https` scheme and a non-empty host. If you add a new registry-URL entry point (new plugin, new step), call this validator rather than trusting the string as-is.
- Default/fallback registry values (`localhost:5000`, empty string) exist in `buildImageRefFromConfig` (`internal/plugins/nomad_deploy.go`) for local dev — do not let a production code path silently fall through to those defaults; require an explicit registry configuration for anything that pushes or deploys.

### Plugin loading

- `plugin.Open()` on a `.so` path from config runs arbitrary native code in-process, with the same privileges as the Shipwright process — this is the highest-severity surface in the codebase. Treat the `path` field in a `type: file` plugin config entry as a trust boundary:
  - Never resolve a plugin path that came from a PR-controlled or otherwise untrusted pipeline YAML without an explicit allowlist (a fixed directory the operator controls, not an arbitrary path from the config file).
  - Prefer `LoadBuiltin` (the registered-factory path) over `LoadFromFile` wherever the plugin is known ahead of time; reserve `LoadFromFile` for operator-initiated, out-of-band plugin installation, not something a pipeline config can trigger unattended.
  - If you need to broaden what `LoadFromFile` accepts, add path validation (must resolve inside an allowed plugins directory) alongside the existing `filepath.Abs` + existence check — those two checks alone do not constrain *which* file gets loaded, only that some file exists at the resolved path.

### Command construction from config/YAML values

- Prefer an argv-array `exec.CommandContext(ctx, "tool", "arg1", "arg2")` (as `internal/executors/native_executor.go` does throughout) over building a shell string. `internal/plugins/nomad_deploy.go` builds `fmt.Sprintf("echo '%s' > /tmp/job.hcl", nomadJob)` and runs it via `sh -c`; a `nomadJob` value containing a single quote breaks out of the intended command. When touching that code path (or writing a similar one), replace the interpolated shell string with either an argv-array `WithExec` call or a file written through `dagger.Directory`/`WithNewFile`, not string interpolation into `sh -c`.
- Any value that reaches `WithExec([]string{"sh", "-c", ...})` should be treated as a shell-injection candidate if it can trace back to YAML/plugin config or an environment variable outside this process's own control.
- `internal/config/yaml_parser.go` unmarshals into typed structs via `gopkg.in/yaml.v3` — this is safe from YAML-triggered code execution (no custom unmarshaler shells out), so the risk is downstream consumption of the parsed strings, not the parsing step itself. Validate/allowlist values (step names, registry URLs) at the point of *use*, following the existing `ValidateConfig`/`ValidateRegistryURL` pattern, not just at parse time.

## Review Checklist

- [ ] Any new credential is wrapped in `dagger.Secret` before crossing into a container or command, never passed as a plain string.
- [ ] No new `String()`/logging path prints a raw secret value.
- [ ] Any new registry URL entry point calls `ValidateRegistryURL`.
- [ ] Any new plugin-loading path either uses `LoadBuiltin` or restricts `LoadFromFile` to an operator-controlled directory.
- [ ] No new `fmt.Sprintf`-built shell string embeds a config/YAML/env-sourced value inside `sh -c`; use argv-array `WithExec`/`exec.Command` instead.
- [ ] SDD note: if the change alters how a public pipeline contract accepts secrets or credentials (e.g. a new provider YAML field for auth), it needs an `openspec` change, not just this review.

## Commands

```bash
# Find places that build shell strings from formatted values
rg -n 'sh", "-c"' --glob '*.go'

# Find places that call plugin.Open / LoadFromFile
rg -n 'plugin\.Open|LoadFromFile' --glob '*.go'

# Find places that pass secrets into containers
rg -n 'SetSecret|WithRegistryAuth' --glob '*.go'
```
