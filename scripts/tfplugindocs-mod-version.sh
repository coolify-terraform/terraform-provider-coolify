#!/usr/bin/env bash
# Print the terraform-plugin-docs version from a go.mod (no leading v).
# Handles single-line require and block-style require ( lines.
# Usage: tfplugindocs-mod-version.sh <go.mod>
set -eu

file="${1:-}"
if [ -z "$file" ]; then
	echo "usage: $0 <go.mod>" >&2
	exit 2
fi
if [ ! -f "$file" ]; then
	echo "FAIL: go.mod not found: $file" >&2
	exit 1
fi

# Scan every field for a v-prefixed semver. Field 2 is the module path
# on a single-line "require github.com/.../terraform-plugin-docs vX.Y.Z".
awk '/terraform-plugin-docs / {
	for (i = 1; i <= NF; i++) {
		if ($i ~ /^v[0-9]/) {
			sub(/^v/, "", $i)
			print $i
			exit
		}
	}
}' "$file"
