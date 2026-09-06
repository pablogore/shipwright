# OpenSpec — Source-of-Truth Specs

This directory holds the merged, canonical specs for `shipwright`, organized by
domain: `openspec/specs/{domain}/spec.md`.

Specs here are updated only by `sdd-archive`, which merges delta specs from
`openspec/changes/{change-name}/specs/` into the matching domain spec once a change is
archived. Do not edit files here directly while a change is in flight — edit the delta
spec under the active change folder instead.

No domain specs exist yet; this repository was just bootstrapped for SDD. The first
`sdd-spec` run for a change will create the first domain subdirectory here after archive.
