#!/usr/bin/env bash
# Emit Go package import paths that contain TestAcc tests for one CI
# acceptance-test shard.
#
# Usage:
#   scripts/ci-acc-packages.sh <shard_index> <shard_count>
#
# Design:
# - Only packages under ./internal/service/... with a TestAcc function.
#   Matches the existing CI gotestsum --packages ./internal/service/... filter.
# - Pin the heaviest acc packages onto different shards first.
# - Round-robin the rest so new packages do not all land on shard 0.
#
# Exit non-zero if a shard would be empty (misconfiguration).
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <shard_index> <shard_count>" >&2
  exit 2
fi

shard="$1"
count="$2"

if ! [[ "$shard" =~ ^[0-9]+$ && "$count" =~ ^[0-9]+$ ]]; then
  echo "error: shard_index and shard_count must be non-negative integers" >&2
  exit 2
fi
if (( count < 1 || shard < 0 || shard >= count )); then
  echo "error: need 0 <= shard_index < shard_count (got shard=$shard count=$count)" >&2
  exit 2
fi

# Heaviest acceptance packages by TestAcc surface and Coolify work
# (application CRUD + integration, compose services, then databases).
# Only the first $count existing entries are exclusive pins.
heavy_pins=(
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/service"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/postgresql"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/environmentvariable"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/deployment"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/scheduledtask"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/githubapp"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/storage"
)

all_pkgs_file="$(mktemp)"
acc_pkgs_file="$(mktemp)"
selected_file="$(mktemp)"
assigned_file="$(mktemp)"
trap 'rm -f "$all_pkgs_file" "$acc_pkgs_file" "$selected_file" "$assigned_file"' EXIT

go list ./internal/service/... | LC_ALL=C sort >"$all_pkgs_file"
module="$(go list -m)"

pkg_has_testacc() {
  local pkg="$1"
  local rel="${pkg#"$module"/}"
  local files=()
  local f
  # nullglob: packages without _test.go must not leave a literal glob.
  shopt -s nullglob
  files=("$rel"/*_test.go)
  shopt -u nullglob
  if (( ${#files[@]} == 0 )); then
    return 1
  fi
  for f in "${files[@]}"; do
    if grep -qE '^func TestAcc' "$f"; then
      return 0
    fi
  done
  return 1
}

while IFS= read -r pkg; do
  if pkg_has_testacc "$pkg"; then
    printf '%s\n' "$pkg" >>"$acc_pkgs_file"
  fi
done <"$all_pkgs_file"

if [[ ! -s "$acc_pkgs_file" ]]; then
  echo "error: no TestAcc packages found under ./internal/service/..." >&2
  exit 1
fi

pkg_in_acc() {
  grep -Fxq "$1" "$acc_pkgs_file"
}

# 1) Exclusive pin: first `count` existing heavies, one per shard.
pin_i=0
for pkg in "${heavy_pins[@]}"; do
  if ! pkg_in_acc "$pkg"; then
    continue
  fi
  if (( pin_i >= count )); then
    break
  fi
  printf '%s\n' "$pkg" >>"$assigned_file"
  if (( pin_i == shard )); then
    printf '%s\n' "$pkg" >>"$selected_file"
  fi
  pin_i=$((pin_i + 1))
done

# 2) Remaining acc packages: stable sorted order, round-robin into shards.
rest_i=0
while IFS= read -r pkg; do
  if grep -Fxq "$pkg" "$assigned_file" 2>/dev/null; then
    continue
  fi
  target=$((rest_i % count))
  rest_i=$((rest_i + 1))
  if (( target == shard )); then
    printf '%s\n' "$pkg" >>"$selected_file"
  fi
done <"$acc_pkgs_file"

if [[ ! -s "$selected_file" ]]; then
  echo "error: shard $shard of $count selected zero packages" >&2
  exit 1
fi

cat "$selected_file"
