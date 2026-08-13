#!/usr/bin/env bash
# Emit go package import paths for one CI unit-test shard.
#
# Usage:
#   scripts/ci-unit-packages.sh <shard_index> <shard_count>
#
# Design:
# - Pin the heaviest packages (from CI timings) onto different shards first so
#   the wall-clock critical path is closer to max(package) than sum(packages).
# - Round-robin the remaining packages for balance as the suite grows.
# - Pure packages (no terraform-plugin-testing fork) may run with higher
#   -parallel; callers decide flags. This script only selects packages.
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

# Heaviest UnitTest packages by recent CI package wall time.
# Only the first $count entries are exclusive pins (one per shard). Remaining
# names fall through to the general round-robin so application does not also
# carry scheduledtask+storage+deployment on a 3-wide matrix (PR #729 Test(0)
# was 10.5m vs ~6m for the other shards).
#
# On 4+ shards, application is also added to shard 1. CI compiles complementary
# test-file slices with -tags=ci_app_a (shard 0) and -tags=ci_app_b (shard 1)
# so the ~10m application UnitTest package is no longer a single-job floor.
APP_PKG="github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
heavy_pins=(
  "$APP_PKG"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/service"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/redis"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/scheduledtask"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/githubapp"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/environmentvariable"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/storage"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/volumebackup"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/project"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/deployment"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/backup"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/privatekey"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/server"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/s3storage"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/cloudtoken"
  "github.com/coolify-terraform/terraform-provider-coolify/internal/service/environment"
)

all_pkgs_file="$(mktemp)"
selected_file="$(mktemp)"
assigned_file="$(mktemp)"
trap 'rm -f "$all_pkgs_file" "$selected_file" "$assigned_file"' EXIT

go list ./... | grep -v '/tools' | LC_ALL=C sort >"$all_pkgs_file"

pkg_in_module() {
  grep -Fxq "$1" "$all_pkgs_file"
}

# 1) Exclusive pin: first `count` existing heavies, one per shard.
pin_i=0
for pkg in "${heavy_pins[@]}"; do
  if ! pkg_in_module "$pkg"; then
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

# 2b) Split application across shards 0 and 1 when the matrix is wide enough.
# Shard 0 already has it from the exclusive pin. Shard 1 runs the complementary
# test files (see ci.yml -tags=ci_app_b) in the same gotestsum as service.
if (( count >= 4 && shard == 1 )); then
  if pkg_in_module "$APP_PKG"; then
    printf '%s\n' "$APP_PKG" >>"$selected_file"
  fi
fi

# 2) Remaining packages: stable sorted order, round-robin into shards.
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
done <"$all_pkgs_file"

if [[ ! -s "$selected_file" ]]; then
  echo "error: shard $shard of $count selected zero packages" >&2
  exit 1
fi

cat "$selected_file"
