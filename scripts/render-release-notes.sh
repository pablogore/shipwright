#!/usr/bin/env bash
set -euo pipefail

# Composes the GitHub Release body from the same generated changelog that
# feeds CHANGELOG.md (see changelog-prepend-release.sh) -- so the release
# body and CHANGELOG.md are always exactly the same content, never two
# independently-generated changelogs that can drift apart (RELEASE-DIST-01B
# PR3).
#
# <changelog-file> holds commit-message-derived text, which is untrusted --
# it is read from file and concatenated, never interpolated into a shell
# command, matching changelog-prepend-release.sh's handling of the same
# content.
#
# Usage: scripts/render-release-notes.sh <tag> <changelog-file> <output-file>

TAG="${1:?tag required}"
CHANGELOG_FILE="${2:?changelog file required}"
OUTPUT_FILE="${3:?output file required}"
VERSION="${TAG#v}"

{
  cat "$CHANGELOG_FILE"
  echo ""
  echo "---"
  echo ""
  cat <<EOF
## 📦 Installation

Download the appropriate binary for your platform and architecture.

### Direct Binary Download (Recommended for CI/CD)

Download the binary directly without extracting:

\`\`\`bash
# Linux/macOS
curl -L https://github.com/pablogore/shipwright/releases/download/${TAG}/shipwright-\$(uname -s | tr '[:upper:]' '[:lower:]')-\$(uname -m | sed 's/x86_64/amd64/') -o shipwright
chmod +x shipwright
sudo mv shipwright /usr/local/bin/
\`\`\`

Available binaries:
- \`shipwright-linux-amd64\` - Linux AMD64
- \`shipwright-linux-arm64\` - Linux ARM64
- \`shipwright-darwin-amd64\` - macOS AMD64
- \`shipwright-darwin-arm64\` - macOS ARM64
- \`shipwright-windows-amd64.exe\` - Windows AMD64
- \`shipwright-windows-arm64.exe\` - Windows ARM64

### Archive Download

Or download the full archive:

\`\`\`bash
# Linux/macOS
curl -L https://github.com/pablogore/shipwright/releases/download/${TAG}/shipwright_${VERSION}_\$(uname -s | tr '[:upper:]' '[:lower:]')_\$(uname -m | sed 's/x86_64/amd64/').tar.gz | tar xz
sudo mv shipwright /usr/local/bin/

# Windows
# Download the .zip file and extract shipwright.exe
\`\`\`

### For CI/CD Usage

Services can download and use the binary directly in their pipelines:

\`\`\`yaml
- name: Download Shipwright
  run: |
    curl -L https://github.com/pablogore/shipwright/releases/download/${TAG}/shipwright-linux-amd64 -o shipwright
    chmod +x shipwright
\`\`\`
EOF
} > "$OUTPUT_FILE"

echo "✅ Release notes rendered to ${OUTPUT_FILE}"
