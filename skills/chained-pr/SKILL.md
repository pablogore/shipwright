---
name: shipwright-chained-pr
description: "Trigger: PRs over 400 lines, stacked PRs, review slices. Split oversized Shipwright changes into chained PRs on Git Flow branches."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Load this skill when a planned PR may exceed **400 changed lines**, an `openspec` change forecasts a large or multi-contract diff, or the user asks for chained/stacked PRs, review slices, or reviewer-load control.

## Hard Rules

- Split PRs over **400 changed lines** unless a maintainer explicitly accepts an exception.
- Keep each PR reviewable in about **≤60 minutes**.
- Use one deliverable work unit per PR; keep tests with the behavior they verify.
- All chain branches are cut from `develop` (Git Flow), never from `main`, unless the chain is delivering a hotfix.
- **A PR touching a public pipeline contract (provider YAML shape, step interfaces under `internal/interfaces/`, plugin contracts) needs an `openspec` change backing it.** When slicing a chain, put the contract-defining slice first and reference the `openspec/changes/<change-id>/` folder in every PR of that chain, so reviewers can see the change is already spec-approved before code lands.
- State start, end, prior dependencies, follow-up work, and out-of-scope items in every chained PR.
- Every child PR must include a dependency diagram marking the current PR with `📍`.
- In Feature Branch Chain, create a draft/no-merge tracker PR based on `develop`; child PR #1 targets the tracker branch, later children target the immediate parent branch.
- Treat polluted diffs as base bugs: retarget or rebase until only the current work unit appears.
- Do not mix chain strategies after the user chooses one.

## Decision Gates

| Condition | Action |
|---|---|
| PR ≤400 changed lines and focused | Keep single PR. |
| PR >400, each slice can land independently on `develop` | Use Stacked PRs to `develop`. |
| PR >400, feature must integrate before `develop` | Use Feature Branch Chain with a tracker PR into `develop`. |
| PR changes a public pipeline contract | Land (or reference) the backing `openspec` change before/alongside the first slice. |
| Generated/vendor/migration diff cannot split cleanly | Ask maintainer for an explicit size exception. |

## Execution Steps

1. Estimate changed lines and identify independent work units.
2. Check whether the change touches a public pipeline contract; if so, confirm the `openspec` change exists or is being created alongside slice 1.
3. Ask for a chain strategy when none is cached and the budget is exceeded.
4. Create branches/PRs from `develop` using the chosen strategy only.
5. Add a Chain Context section to each PR.
6. Verify each PR independently: tests, local/CI parity, rollback scope, and clean diff.
7. Keep the tracker PR draft/no-merge until all child PRs are reviewed and integrated.

## Chain Context Section

```markdown
## Chain Context

| Field | Value |
|-------|-------|
| Chain | <feature or stack name> |
| Tracker PR | <#NNN or "Not needed"> |
| Position | <N of total> |
| Base | `<target branch>` |
| Openspec change | `<openspec/changes/<change-id>/> or "N/A"` |
| Depends on | <PR/issue/link or "None"> |
| Follow-up | <next PR or "None"> |
| Review budget | <changed lines> / 400 |

### Chain Overview

\`\`\`text
develop
 └── #NNN Previous PR
      └── 📍 #NNN This PR
           └── #NNN Next PR
\`\`\`

### Scope
- Includes: <focused unit>
- Excludes: <deferred work>
```

## Output Contract

Return the chosen strategy, PR order, current PR boundary, dependency diagram, review budget (`additions + deletions`), verification plan, and the linked `openspec` change when the chain touches a public pipeline contract.

## Commands

```bash
gh pr view <PR_NUMBER> --json additions,deletions,changedFiles,title,url
gh pr create --base develop --title "feat(scope): focused slice" --body-file pr-body.md
gh pr create --base feature/my-feature-01-core --title "feat(scope): next focused slice" --body-file pr-body.md
```
