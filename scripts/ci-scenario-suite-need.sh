#!/usr/bin/env bash
# Decide whether a Scenario Tests matrix suite should boot Coolify.
# Writes a single GITHUB_OUTPUT line: run=true or run=false.
# Usage: ci-scenario-suite-need.sh SUITE
# Env (first non-empty wins):
#   github-cicd: GITHUB_APP_ID, COOLIFY_GITHUB_APP_APP_ID, TF_VAR_github_app_id
#   hetzner:     HETZNER_TOKEN, COOLIFY_HETZNER_TOKEN, TF_VAR_hetzner_api_token
# core and all always run.
set -eu

suite="${1:-}"
if [ -z "$suite" ]; then
  echo "FAIL: usage: $0 all|core|github-cicd|hetzner" >&2
  exit 2
fi

github_id="${GITHUB_APP_ID:-${COOLIFY_GITHUB_APP_APP_ID:-${TF_VAR_github_app_id:-}}}"
hetzner_token="${HETZNER_TOKEN:-${COOLIFY_HETZNER_TOKEN:-${TF_VAR_hetzner_api_token:-}}}"

case "$suite" in
  all|core)
    echo "run=true"
    ;;
  github-cicd)
    if [ -n "$github_id" ]; then
      echo "run=true"
    else
      echo "run=false"
    fi
    ;;
  hetzner)
    if [ -n "$hetzner_token" ]; then
      echo "run=true"
    else
      echo "run=false"
    fi
    ;;
  *)
    echo "FAIL: unknown suite ${suite}" >&2
    exit 1
    ;;
esac
