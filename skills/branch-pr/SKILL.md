---
name: shipwright-branch-pr
description: "Create Shipwright pull requests on Git Flow branches with issue-first discipline. Trigger: creating, opening, or preparing PRs for review."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## When to Use

Use this skill when:
- Creating a pull request for any change to this repository
- Preparing a branch for submission
- Helping a contributor open a PR against `develop` or `main`

---

## Critical Rules

1. **This repo uses Git Flow, not GitHub Flow** — branch from `develop`, never commit directly to `main` or `develop`.
2. **Link the driving issue in the PR body** (`Closes #N` / `Fixes #N` / `Resolves #N`) whenever an issue exists — this repo does not currently run automated issue-link enforcement, so treat it as required discipline, not a gate you can rely on CI to catch.
3. **A PR touching a public pipeline contract requires an SDD change backing it** (see `openspec/changes/`). If the diff changes provider YAML shape, step contracts, or plugin interfaces under `internal/interfaces/`, confirm an approved `openspec` change exists before opening the PR.
4. **Hotfixes target `main` directly** and bypass `develop`; because of that, CI runs full validation on hotfix branches (see `.github/workflows/ci.yml`). Everything else targets `develop`.
5. **No `Co-Authored-By` / AI attribution trailers** in commits or PR descriptions — conventional commits only.

---

## Branch Naming (Git Flow)

Branch from `develop` (or `main` for hotfixes only):

| Type | Branch pattern | Base | Example |
|------|---------------|------|---------|
| Feature | `feature/<description>` | `develop` | `feature/plugin-retry-policy` |
| Bug fix | `fix/<description>` | `develop` | `fix/yaml-step-parsing` |
| Hotfix | `hotfix/<description>` | `main` | `hotfix/registry-auth-panic` |
| Chore | `chore/<description>` | `develop` | `chore/bump-dagger-sdk` |

Format: `type/description` — lowercase, hyphen-separated, no spaces.

```bash
# Feature or fix
git checkout develop
git pull origin develop
git checkout -b feature/my-change

# Hotfix (urgent production fix, bypasses develop)
git checkout main
git pull origin main
git checkout -b hotfix/critical-bug
```

---

## Conventional Commits

Commit messages should follow:

```
type(scope): description
```

`type` — one of: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`. Use `!` after the type for a breaking change (`feat!: ...`).

```
feat(plugins): add retry policy to nomad-deploy hook
fix(config): validate registry URL scheme before use
docs(pipeline-development): document provider YAML contract
chore(deps): upgrade dagger go sdk
```

---

## PR Body Checklist

There is no repository-provided `PULL_REQUEST_TEMPLATE.md` in this repo, so build the body from these sections:

1. **Linked issue** — `Closes #N` when an issue exists.
2. **Summary** — 1-3 bullets of what changed and why.
3. **Changes** — table of touched paths and what changed in each.
4. **SDD reference** — if the PR implements or completes an `openspec` change, link the change folder under `openspec/changes/<change-id>/`.
5. **Test plan** — exact commands run locally (see the `shipwright-testing` skill) and their result, including local/CI parity confirmation when Dagger-executed steps are involved.
6. **Rollback** — what reverting this PR removes, and whether it's isolated.

---

## Commands

```bash
# Branch from develop
git checkout develop && git pull origin develop
git checkout -b feature/my-feature

# Push and open PR against develop
git push -u origin feature/my-feature
gh pr create --base develop --title "feat(scope): description" --body "Closes #N"

# Hotfix against main
git checkout main && git pull origin main
git checkout -b hotfix/urgent-fix
git push -u origin hotfix/urgent-fix
gh pr create --base main --title "fix(scope): description"
```
