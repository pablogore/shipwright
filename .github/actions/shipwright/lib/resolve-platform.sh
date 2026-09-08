#!/usr/bin/env bash
# Normalizes and validates the runner's OS/arch into Shipwright's actual
# release asset naming (docs/RELEASE_INTEGRITY_CONTRACT.md's "Distribution
# identity": shipwright-<os>-<arch>, with a .exe suffix on Windows). Fails
# closed on anything outside the 6 combos release.yml actually publishes --
# never silently maps an unrecognized platform to a default one.
#
# Usage: resolve-platform.sh <uname-s-output> <uname-m-output>
# On success, appends os=/arch=/binary= to $GITHUB_OUTPUT.
set -euo pipefail

OS_RAW="${1:?OS argument required}"
ARCH_RAW="${2:?ARCH argument required}"

case "$(echo "$OS_RAW" | tr '[:upper:]' '[:lower:]')" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  msys*|mingw*|cygwin*|windows*) OS="windows" ;;
  *)
    echo "::error::Unsupported OS '${OS_RAW}'. Shipwright publishes binaries only for linux, darwin, and windows -- see docs/RELEASE_INTEGRITY_CONTRACT.md." >&2
    exit 1
    ;;
esac

case "$(echo "$ARCH_RAW" | tr '[:upper:]' '[:lower:]')" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "::error::Unsupported architecture '${ARCH_RAW}'. Shipwright publishes binaries only for amd64 and arm64 -- see docs/RELEASE_INTEGRITY_CONTRACT.md." >&2
    exit 1
    ;;
esac

BINARY="shipwright-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY="${BINARY}.exe"
fi

{
  echo "os=${OS}"
  echo "arch=${ARCH}"
  echo "binary=${BINARY}"
} >> "$GITHUB_OUTPUT"
