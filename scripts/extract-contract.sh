#!/usr/bin/env bash
# Extract API contract from a Coolify release tag.
#
# Usage:
#   scripts/extract-contract.sh v4.0.1
#   scripts/extract-contract.sh latest    # uses HEAD of main branch
#
# Clones the specified Coolify version to a temp directory, runs the
# Python extraction script, and saves the contract JSON to testdata/contracts/.
# If a previous contract exists, prints a summary diff.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONTRACTS_DIR="$REPO_ROOT/testdata/contracts"

VERSION="${1:-latest}"

if ! command -v python3 >/dev/null 2>&1; then
    echo "ERROR: python3 is required to run scripts/extract-contract.sh. Install Python 3.9+ and re-run." >&2
    exit 1
fi

CLONE_DIR="$(mktemp -d)"
trap 'rm -rf "$CLONE_DIR"' EXIT

echo "==> Cloning coollabsio/coolify@${VERSION}..."
if [ "$VERSION" = "latest" ]; then
    git clone --depth 1 https://github.com/coollabsio/coolify.git "$CLONE_DIR/coolify" 2>/dev/null
    VERSION="v4-latest"
else
    git clone --depth 1 --branch "$VERSION" https://github.com/coollabsio/coolify.git "$CLONE_DIR/coolify" 2>/dev/null
fi

OUTPUT="$CONTRACTS_DIR/coolify-${VERSION}.json"
mkdir -p "$CONTRACTS_DIR"

echo "==> Extracting contract..."
python3 "$SCRIPT_DIR/extract-contract.py" "$CLONE_DIR/coolify" \
    --version "$VERSION" \
    --output "$OUTPUT"

echo "==> Contract saved to $OUTPUT"

# Find previous version for diff
PREVIOUS=$(ls -1 "$CONTRACTS_DIR"/coolify-*.json 2>/dev/null | grep -v "$OUTPUT" | sort | tail -1 || true)
if [ -n "$PREVIOUS" ] && [ -f "$PREVIOUS" ]; then
    echo ""
    echo "==> Diff vs $(basename "$PREVIOUS"):"
    "$SCRIPT_DIR/diff-contracts.sh" "$PREVIOUS" "$OUTPUT" || true
fi
