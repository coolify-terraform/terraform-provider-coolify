#!/usr/bin/env bash
# Run terraform test for selected examples/scenarios/acme-* directories.
# Suites (SCENARIO_SUITE or --suite):
#   all          every acme-* dir (local default)
#   core         all except acme-github-cicd and acme-hetzner-infra
#   github-cicd  acme-github-cicd only
#   hetzner      acme-hetzner-infra only
# Retries acme-github-cicd once (Coolify deploy status "failed" flake, #728).
# On any failure, dumps Coolify deployment JSON when endpoint/token are set.
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIAG_DIR="${SCENARIO_DIAG_DIR:-/tmp/scenario-diag}"
RETRY_SCENARIO="${SCENARIO_RETRY:-acme-github-cicd}"
SUITE="${SCENARIO_SUITE:-all}"
LIST_ONLY=0

usage() {
  echo "usage: $0 [--list] [--suite all|core|github-cicd|hetzner]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --list)
      LIST_ONLY=1
      shift
      ;;
    --suite)
      if [[ $# -lt 2 ]]; then
        usage
        exit 2
      fi
      SUITE="$2"
      shift 2
      ;;
    --suite=*)
      SUITE="${1#--suite=}"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "FAIL: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

list_scenario_dirs() {
  suite="$1"
  case "$suite" in
    all|core|github-cicd|hetzner) ;;
    *)
      echo "FAIL: unknown suite '$suite' (want all, core, github-cicd, hetzner)" >&2
      return 2
      ;;
  esac
  for dir in "${ROOT}"/examples/scenarios/acme-*/; do
    [ -d "$dir" ] || continue
    scenario="$(basename "$dir")"
    case "$suite" in
      all) echo "$dir" ;;
      core)
        case "$scenario" in
          acme-github-cicd|acme-hetzner-infra) ;;
          *) echo "$dir" ;;
        esac
        ;;
      github-cicd)
        if [ "$scenario" = "acme-github-cicd" ]; then
          echo "$dir"
        fi
        ;;
      hetzner)
        if [ "$scenario" = "acme-hetzner-infra" ]; then
          echo "$dir"
        fi
        ;;
    esac
  done
}

if [ "$LIST_ONLY" -eq 1 ]; then
  list_scenario_dirs "$SUITE" || exit $?
  exit 0
fi

SCENARIO_LIST="$(list_scenario_dirs "$SUITE")" || exit $?
if [ -z "$SCENARIO_LIST" ]; then
  echo "FAIL: suite '$SUITE' selected zero scenario directories" >&2
  exit 2
fi
SCENARIO_COUNT="$(printf '%s\n' "$SCENARIO_LIST" | grep -c .)"

echo "PLAN: terraform test suite=${SUITE} (${SCENARIO_COUNT} dir(s); retry ${RETRY_SCENARIO} once on failure)"

dump_diag() {
  endpoint="${COOLIFY_ENDPOINT:-${TF_VAR_coolify_endpoint:-}}"
  token="FAKESECRET_e2f3g4h5i6j7k8l9m0n1"
  if [ -z "$endpoint" ] || [ -z "$token" ]; then
    echo "WAIT: skip deployment dump (COOLIFY_ENDPOINT or COOLIFY_TOKEN unset)"
    return 0
  fi
  echo "DO: dump Coolify deployments to ${DIAG_DIR}"
  python3 "${ROOT}/scripts/dump-coolify-deployments.py" \
    --endpoint "$endpoint" \
    --token "$token" \
    --out-dir "$DIAG_DIR" || echo "FAIL: dump-coolify-deployments.py exited $?"
}

run_one() {
  dir="$1"
  (
    cd "$dir" || exit 1
    if [ -d "modules" ]; then
      echo "DO: terraform get in ${dir}"
      terraform get
    fi
    echo "DO: terraform test -verbose in ${dir}"
    terraform test -verbose
  )
}

failed=0
while IFS= read -r dir; do
  [ -d "$dir" ] || continue
  scenario="$(basename "$dir")"
  echo "=== Testing ${scenario} ==="

  case "$scenario" in
    acme-github-cicd)
      if [ -z "${TF_VAR_github_app_id:-}" ]; then
        echo "SKIP: GitHub App secrets not configured"
        echo ""
        continue
      fi
      ;;
    acme-hetzner-infra)
      if [ -z "${TF_VAR_hetzner_api_token:-}" ]; then
        echo "SKIP: Hetzner token not configured"
        echo ""
        continue
      fi
      ;;
  esac

  if run_one "$dir"; then
    echo "OK: ${scenario}"
  else
    if [ "$scenario" = "$RETRY_SCENARIO" ]; then
      echo "DO: retry ${scenario} once (Coolify deploy flake; see #728)"
      if run_one "$dir"; then
        echo "OK: ${scenario} after retry"
      else
        echo "FAIL: ${scenario} after retry"
        dump_diag
        failed=1
      fi
    else
      echo "FAIL: ${scenario}"
      dump_diag
      failed=1
    fi
  fi
  echo ""
done <<< "$SCENARIO_LIST"

if [ "$failed" -eq 0 ]; then
  echo "DONE: all scenario tests passed (suite=${SUITE})"
  echo "NEXT: none"
  exit 0
fi
echo "DONE: one or more scenario tests failed (suite=${SUITE}; diag: ${DIAG_DIR})"
echo "NEXT: inspect terraform test output and ${DIAG_DIR}"
exit 1
