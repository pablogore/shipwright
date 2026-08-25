# Shipwright — Agent Skills Index

When working on this project, load the relevant skill(s) BEFORE writing any code.

Naming convention: `shipwright-*` skills are repo-specific workflow skills.
Unprefixed skills are portable writing or work-unit skills.
`sdd-*` skills drive the Spec-Driven Development cycle.

## Project Context

- Product: [`docs/PRD.md`](docs/PRD.md)
- Engineering architecture: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- Canonical specs: [`openspec/specs/`](openspec/specs/)

## How to Use

1. Check the trigger column
2. Read the referenced SKILL.md
3. Follow all rules from the loaded skill
4. Multiple skills may apply simultaneously

## Skills

| Skill | Trigger | Path |
|---|---|---|
| `shipwright-security` | Security-sensitive changes, secrets, registries, external inputs. | [`skills/security/SKILL.md`](skills/security/SKILL.md) |
| `shipwright-testing` | Tests, mocks, test placement, CI validation. | [`skills/testing/SKILL.md`](skills/testing/SKILL.md) |
| `shipwright-issue-creation` | Creating bugs/features/issues. | [`skills/issue-creation/SKILL.md`](skills/issue-creation/SKILL.md) |
| `shipwright-branch-pr` | Creating/preparing PRs. | [`skills/branch-pr/SKILL.md`](skills/branch-pr/SKILL.md) |
| `shipwright-chained-pr` | Large work split into stacked PRs. | [`skills/chained-pr/SKILL.md`](skills/chained-pr/SKILL.md) |
| `cognitive-doc-design` | Writing architecture/product docs. | [`skills/cognitive-doc-design/SKILL.md`](skills/cognitive-doc-design/SKILL.md) |
| `comment-writer` | PR comments / issue responses. | [`skills/comment-writer/SKILL.md`](skills/comment-writer/SKILL.md) |
| `work-unit-commits` | Splitting implementation into reviewable commits. | [`skills/work-unit-commits/SKILL.md`](skills/work-unit-commits/SKILL.md) |
| `sdd-explore` | Explore before committing to a change. | `~/.claude/skills/sdd-explore/SKILL.md` |
| `sdd-init` | Initialize SDD context. | `~/.claude/skills/sdd-init/SKILL.md` |
| `sdd-apply` | Implement approved SDD tasks. | `~/.claude/skills/sdd-apply/SKILL.md` |
| `sdd-verify` | Verify implementation against spec/design/tasks. | `~/.claude/skills/sdd-verify/SKILL.md` |
| `sdd-archive` | Archive completed changes. | `~/.claude/skills/sdd-archive/SKILL.md` |
| `judgment-day` | Adversarial/blind review before merge. | `~/.claude/skills/judgment-day/SKILL.md` |
