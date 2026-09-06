---
name: shipwright-issue-creation
description: "Create and triage GitHub issues for Shipwright from repository evidence. Trigger: issue creation, bug reports, feature requests, or issue approval."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

# Issue Creation

## When To Use

Use this skill when creating, drafting, triaging, or approving an issue in the Shipwright repository (`github.com/pablogore/shipwright`).

## Core Rule

Discover the repository's actual contribution workflow before proposing or publishing an issue. Do not assume templates, labels, or approval gates exist — verify them. As of this writing, Shipwright has **no `.github/ISSUE_TEMPLATE/` directory and no `CONTRIBUTING.md`**, so the no-template fallback below is the expected path; re-check with the discovery commands each time, since that can change.

## Safe Discovery

Run read-only checks first:

```bash
gh auth status
REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
REPO_URL="$(gh repo view --json url -q .url)"
HOST="${REPO_URL#*://}"
HOST="${HOST%%/*}"
gh repo view --json nameWithOwner,url,hasDiscussionsEnabled,hasIssuesEnabled,isBlankIssuesEnabled
git ls-files CONTRIBUTING.md CONTRIBUTING.* .github/CONTRIBUTING.md .github/ISSUE_TEMPLATE
gh api --hostname "$HOST" --paginate "repos/$REPO/labels?per_page=100" --jq '.[].name'
```

Also inspect:

- `docs/` (Shipwright keeps most contributor-facing process docs here, e.g. `docs/PIPELINE_DEVELOPMENT.md`, `docs/BRANCHING_STRATEGY.md`) and `README.md`;
- files under `.github/ISSUE_TEMPLATE`, if any exist by the time you run this;
- existing open and closed issues for duplicates and established wording;
- `.github/CODEOWNERS` to identify who to mention for review-relevant areas (workflows, docs, scripts).

Stop and ask for repository context if authentication, repository resolution, verification that REPO and HOST are non-empty, required metadata is unavailable, `hasIssuesEnabled` is false, or policy discovery fails. Never continue from failed discovery into issue publication.

A no-template fallback is allowed only when `isBlankIssuesEnabled` is explicitly true. Otherwise follow discovered contact links or stop and ask; never publish.

After discovery and review, build optional label arguments using only labels that exist and repository policy permits the actor to apply:

```bash
LABEL_ARGS=()
# Repeat for each reviewed, permitted discovered label.
LABEL_ARGS+=(--label "$LABEL")
```

An empty array applies no label; do not invent labels.

## Workflow

1. Describe the problem or request in one sentence and derive a short search query.
2. Search open and closed issues:

   ```bash
   gh issue list --repo "$HOST/$REPO" --state all --search "$QUERY" --limit 1000
   ```

   If 1000 results are returned or completeness remains uncertain, narrow the search, use read-only API discovery, or stop and ask before publishing.

3. If an issue already covers the same behavior, comment there instead of creating a duplicate.
4. Choose a repository-provided template only when its purpose matches the report and one exists.
5. Fill every required field from known evidence — repo layout, `internal/` package involved, relevant `docs/*.md`, or `openspec/` change if the issue concerns a public pipeline contract. Ask for missing facts rather than inventing them.
6. Apply labels only when they exist and repository guidance establishes who should apply them.
7. Publish only after the title, body, target repository, and selected template or fallback have been reviewed, and the pre-submission privacy review below has passed.

## Pre-submission Privacy Review

Pre-submission privacy review is mandatory. Scan every issue body immediately before `gh issue create`. The scan replaces — never deletes — environment-specific data with explicit placeholders so the reproduction still teaches:

| Category | Replace with | Example (before → after) |
|----------|---------------|---------------------------|
| Private project names | `<project-name>` | `my-private-fork` → `<project-name>` |
| Usernames | `<user>` | `/Users/my-real-username/go/bin` → `/Users/<user>/go/bin` |
| Hostnames | `<hostname>` | `devbox-macbook.local` → `<hostname>` |
| API keys, tokens, passwords | `<token>` / `<password>` | `ghp_abc123...` → `<token>` |
| Registry URLs / internal hosts | `<registry-host>:<port>` | `registry.internal.example:5000` → `<registry-host>:<port>` |

Do NOT redact intentionally public identifiers: tool names (`shipwright`, `dagger`, `go`), package names, public documentation URLs, generic example domains (`example.com`, `localhost`). Keep reproduction structure with placeholders — never redact an example into nothingness.

**Rule of thumb:** if the reader can run the reproduction step after you replace every identifier with its placeholder, the sanitization is correct. If a step becomes impossible because the placeholder consumed a needed value, mark it `<value-required>` and explain in the body what the user should fill in.

## Template Paths

Do not guess a template filename. If multiple templates could apply and repository guidance does not distinguish them, stop and ask which one to use.

- `.yml`/`.yaml` files are GitHub Issue Forms. Do not parse or render their schema. Open the web issue chooser and stop for human completion:

  ```bash
  gh issue create --repo "$HOST/$REPO" --web "${LABEL_ARGS[@]}"
  ```

- `.md` files are Markdown templates. Read the matching template, complete it from known evidence into a reviewed `BODY_FILE`, then publish it:

  ```bash
  gh issue create --repo "$HOST/$REPO" --title "$TITLE" --body-file "$BODY_FILE" "${LABEL_ARGS[@]}"
  ```

## No-Template Fallback

Given Shipwright currently has no issue templates, this is the default path when the repository permits issue creation and `isBlankIssuesEnabled` is explicitly true. Prepare a structured body with these sections:

- problem or requested outcome;
- reproduction or motivating example (include the Dagger/Go/CI command that surfaced it, sanitized per the privacy review above);
- expected behavior;
- actual behavior or current limitation;
- environment and relevant evidence (Go version, Dagger SDK version, local vs. CI);
- alternatives or workarounds, when applicable.

Publish the reviewed fallback explicitly:

```bash
gh issue create --repo "$HOST/$REPO" --title "$TITLE" --body "$BODY" "${LABEL_ARGS[@]}"
```

If blank issues are not explicitly enabled, follow discovered contact links or stop and ask. Never publish a no-template fallback.

## Labels And Approval

Treat labels and approval gates as conditional:

- use only labels returned by repository discovery;
- follow contribution guidance for who may apply each label;
- wait when repository policy requires maintainer approval before implementation;
- do not invent a status or priority taxonomy when none is documented.

## Questions And Discussions

Use Discussions only when `hasDiscussionsEnabled` is true and repository guidance routes the question there. Otherwise follow documented support/contact links or ask the user where the question belongs. Never link to another repository's Discussions page.

## Triage Decision

Before approving or closing an issue, verify:

- it describes a concrete bug or scoped improvement rather than an unsupported question;
- it is not a duplicate;
- the report contains enough evidence for an implementation decision;
- the requested behavior is in scope for a Dagger-based CI/CD execution tool;
- if the request changes a public pipeline contract (provider YAML shape, step interfaces, plugin contracts), note that it needs an `openspec` change before implementation, not just an approved issue;
- labels and status changes follow the current repository's policy.

If any point is uncertain, keep the issue in the repository's review state and request the smallest missing evidence.
