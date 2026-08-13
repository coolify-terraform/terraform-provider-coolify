#!/usr/bin/env bash
# Run terraform test for each examples/scenarios/acme-* directory.
# Retries acme-github-cicd once (Coolify deploy status "failed" flake, #728).
# On any failure, dumps Coolify deployment JSON when endpoint/token are set.
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIAG_DIR="${SCENARIO_DIAG_DIR:-/tmp/scenario-diag}"
RETRY_SCENARIO="${SCENARIO_RETRY:-acme-github-cicd}"

echo "PLAN: terraform test each examples/scenarios/acme-* (retry ${RETRY_SCENARIO} once on failure)"

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
for dir in "${ROOT}"/examples/scenarios/acme-*/; do
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
done

if [ "$failed" -eq 0 ]; then
  echo "DONE: all scenario tests passed"
  echo "NEXT: none"
  exit 0
fi
echo "DONE: one or more scenario tests failed (diag: ${DIAG_DIR})"
echo "NEXT: inspect terraform test output and ${DIAG_DIR}"
exit 1
