#!/usr/bin/env bash
# Builds the shipwright CLI's argv, one argument per line, from this
# action's resolved inputs. Printed line-by-line (not as a single string)
# specifically so the caller reads it back into a bash array with mapfile
# and execs the binary as `"$BINARY" "${ARGS[@]}"` -- never through `eval`
# or any other shell re-parsing step. That is what makes a value containing
# spaces, quotes, `;`, `$(...)`, or any other shell metacharacter reach the
# CLI as inert, single-argument data instead of being re-split or executed.
# The one accepted limitation: an argument value containing a literal
# newline cannot round-trip through this line-oriented format.
#
# Reads its input entirely from environment variables (never CLI args), so
# a value never has to survive shell-quoting twice on its way in:
#   WORKFLOW, CONFIG, ENV, COVERAGE, BRANCH, LIST_STEPS, STEP, EXECUTOR,
#   VERBOSE, GIT_REF, GIT_AUTH
# Mirrors the exact flag set and ordering the previous eval-based CMD
# string built, minus eval itself.
set -euo pipefail

printf '%s\n' "--workflow=${WORKFLOW}"
printf '%s\n' "--config=${CONFIG}"
printf '%s\n' "--env=${ENV}"
printf '%s\n' "--coverage=${COVERAGE}"
printf '%s\n' "--branch=${BRANCH}"

if [ "${LIST_STEPS:-false}" = "true" ]; then
  printf '%s\n' "--list-steps"
elif [ -n "${STEP:-}" ]; then
  printf '%s\n' "--step=${STEP}"
fi

if [ -n "${EXECUTOR:-}" ]; then
  printf '%s\n' "--executor=${EXECUTOR}"
fi

if [ "${VERBOSE:-false}" = "true" ]; then
  printf '%s\n' "--verbose"
fi

if [ -n "${GIT_REF:-}" ]; then
  printf '%s\n' "--git-ref=${GIT_REF}"
fi

if [ -n "${GIT_AUTH:-}" ]; then
  printf '%s\n' "--git-auth=${GIT_AUTH}"
fi
