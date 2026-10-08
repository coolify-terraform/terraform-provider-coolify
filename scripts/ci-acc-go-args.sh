#!/usr/bin/env bash
# Print go test flags for one nightly acceptance cell, one flag per line.
#
# Usage:
#   scripts/ci-acc-go-args.sh <coolify_image> [matrix_label]
#
# -parallel=1 only serializes tests inside a package. go test still runs
# several packages at once (default -p = NumCPU). Coolify latest returns
# 429 when the full service suite does that. latest, and the stable-latest
# label, run one package at a time and get a longer timeout.
set -euo pipefail

image="${1:-}"
label="${2:-}"
timeout="40m"

if [[ "$image" == "latest" || "$label" == "stable-latest" ]]; then
  printf '%s\n' "-p" "1"
  timeout="55m"
fi

printf '%s\n' "-count=1" "-timeout=${timeout}" "-parallel=1"
