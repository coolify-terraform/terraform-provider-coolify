#!/usr/bin/env bash
# Decide whether a Scenario Tests matrix suite should boot Coolify.
# Writes a single GITHUB_OUTPUT line: run=true or run=false.
# Usage: ci-scenario-suite-need.sh SUITE
# Env: GITHUB_APP_ID (github-cicd), HETZNER_TOKEN (hetzner). core always runs.
set -eu

suite="${1:-}"
if [ -z "$suite" ]; then
  echo "FAIL: usage: $0 all|core|github-cicd|hetzner" >&2
  exit 2
fi

case "$suite" in
  all|core)
    echo "run=true"
    ;;
  github-cicd)
    if [ -n "${GITHUB_APP_ID:-}" ]; then
      echo "run=true"
    else
      echo "run=false"
    fi
    ;;
  hetzner)
    if [ -n "${HETZNER_TOKEN:-}" ]; then
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
