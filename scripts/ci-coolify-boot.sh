#!/usr/bin/env bash
# Coolify CI bootstrap steps. Designed so image pull can overlap Go/Terraform
# setup in the caller, and SSH setup can overlap the pull.
#
# Usage:
#   scripts/ci-coolify-boot.sh <step>
#
# Steps:
#   prepare         Create data dirs, download compose, write .env + override
#   ssh             Enable root SSH to localhost for Coolify server validation
#   pull            docker compose pull (foreground)
#   pull-bg         Start pull in the background; write /tmp/coolify-pull.exit
#   wait-pull       Wait for pull-bg (or no-op if pull already finished)
#   up              docker compose up -d --remove-orphans (no --pull/--force)
#   wait-ready      Poll http://localhost:8000/register until 200/302
#   prepare-pull-bg prepare + ssh + pull-bg (early job step after checkout)
#
# Env:
#   COOLIFY_DATA_DIR     default /home/runner/coolify-data
#   COOLIFY_IMAGE_TAG    default edge
#   COOLIFY_PULL_DIR     marker dir, default /tmp/coolify-pull
#   COOLIFY_READY_TRIES  default 90 (2s each => 3 minutes)
set -euo pipefail

STEP="${1:-}"
if [[ -z "$STEP" ]]; then
  echo "usage: $0 <prepare|ssh|pull|pull-bg|wait-pull|up|wait-ready|prepare-pull-bg>" >&2
  exit 2
fi

COOLIFY_DATA_DIR="${COOLIFY_DATA_DIR:-/home/runner/coolify-data}"
COOLIFY_IMAGE_TAG="${COOLIFY_IMAGE_TAG:-edge}"
PULL_DIR="${COOLIFY_PULL_DIR:-/tmp/coolify-pull}"
SOURCE_DIR="$COOLIFY_DATA_DIR/source"
READY_TRIES="${COOLIFY_READY_TRIES:-90}"

log() { printf '%s\n' "$1"; }

compose() {
  docker compose --env-file .env \
    -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.override.yml \
    "$@"
}

# GNU timeout on Linux CI; no-op on macOS unless coreutils is installed.
run_limited() {
  local secs="$1"
  shift
  if command -v timeout >/dev/null 2>&1; then
    timeout --foreground "$secs" "$@"
  else
    "$@"
  fi
}

step_prepare() {
  log "PLAN: write Coolify compose files under $SOURCE_DIR (tag=${COOLIFY_IMAGE_TAG})"
  log "DO: create data directories"
  mkdir -p "$COOLIFY_DATA_DIR"/{source,ssh/{keys,mux},applications,databases,backups,services,proxy,webhooks-during-maintenance}
  mkdir -p "$COOLIFY_DATA_DIR/proxy/dynamic"
  mkdir -p "$SOURCE_DIR"
  cd "$SOURCE_DIR"

  log "DO: download compose manifests"
  curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors \
    https://cdn.coollabs.io/coolify/docker-compose.yml -o docker-compose.yml
  curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors \
    https://cdn.coollabs.io/coolify/docker-compose.prod.yml -o docker-compose.prod.yml
  curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors \
    https://cdn.coollabs.io/coolify/.env.production -o .env

  sed -i "s|/data/coolify|$COOLIFY_DATA_DIR|g" docker-compose.yml docker-compose.prod.yml .env
  sed -i "s|APP_ID=.*|APP_ID=$(openssl rand -hex 16)|g" .env
  sed -i "s|APP_KEY=.*|APP_KEY=base64:$(openssl rand -base64 32)|g" .env
  sed -i "s|DB_PASSWORD=.*|DB_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
  sed -i "s|REDIS_PASSWORD=.*|REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
  sed -i "s|PUSHER_APP_ID=.*|PUSHER_APP_ID=$(openssl rand -hex 32)|g" .env
  sed -i "s|PUSHER_APP_KEY=.*|PUSHER_APP_KEY=$(openssl rand -hex 32)|g" .env
  sed -i "s|PUSHER_APP_SECRET=.*|PUSHER_APP_SECRET=$(openssl rand -hex 32)|g" .env
  if grep -qE '^LATEST_IMAGE=' .env; then
    sed -i "s|^LATEST_IMAGE=.*|LATEST_IMAGE=${COOLIFY_IMAGE_TAG}|" .env
  else
    echo "LATEST_IMAGE=${COOLIFY_IMAGE_TAG}" >> .env
  fi
  log "OK: Coolify image tag ${COOLIFY_IMAGE_TAG}"

  if [[ ! -f "$COOLIFY_DATA_DIR/ssh/keys/id.root@host.docker.internal" ]]; then
    ssh-keygen -t ed25519 -f "$COOLIFY_DATA_DIR/ssh/keys/id.root@host.docker.internal" -N "" -q
  fi
  mkdir -p ~/.ssh
  cat "$COOLIFY_DATA_DIR/ssh/keys/id.root@host.docker.internal.pub" >> ~/.ssh/authorized_keys
  if [[ -d /root/.ssh ]]; then
    sudo sh -c "cat $COOLIFY_DATA_DIR/ssh/keys/id.root@host.docker.internal.pub >> /root/.ssh/authorized_keys"
  fi
  printf 'services:\n  coolify:\n    extra_hosts:\n      - "host.docker.internal:host-gateway"\n' \
    > docker-compose.override.yml
  sudo chown -R 9999:9999 "$COOLIFY_DATA_DIR/ssh"
  docker network create --attachable coolify >/dev/null 2>&1 || true
  log "OK: prepare complete"
}

step_ssh() {
  log "PLAN: enable passwordless root SSH on localhost"
  log "DO: start sshd and install runner key"
  sudo systemctl enable ssh
  sudo systemctl start ssh
  if [[ ! -f ~/.ssh/id_ed25519 ]]; then
    ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N "" -q
  fi
  mkdir -p ~/.ssh
  cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
  chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys
  sudo sed -i 's/^#*PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
  sudo systemctl restart ssh
  sudo mkdir -p /root/.ssh
  sudo cp ~/.ssh/authorized_keys /root/.ssh/authorized_keys
  sudo chmod 700 /root/.ssh
  sudo chmod 600 /root/.ssh/authorized_keys
  ssh -o StrictHostKeyChecking=no -o BatchMode=yes root@localhost echo "Root SSH OK"
  log "OK: root SSH works"
}

step_pull() {
  log "PLAN: pull Coolify images (no container recreate)"
  cd "$SOURCE_DIR"
  log "DO: docker compose pull"
  compose pull
  log "OK: images pulled"
}

step_pull_bg() {
  log "PLAN: start Coolify image pull in background"
  mkdir -p "$PULL_DIR"
  rm -f "$PULL_DIR/exit" "$PULL_DIR/log"
  cd "$SOURCE_DIR"
  log "DO: nohup docker compose pull (markers in $PULL_DIR)"
  # Marker file so wait-pull can poll across GitHub Actions steps (wait(1)
  # only works for children of the same shell). $1/$2 expand in the child.
  # shellcheck disable=SC2016
  nohup bash -c '
    set +e
    cd "$1"
    docker compose --env-file .env \
      -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.override.yml \
      pull
    echo $? > "$2/exit"
  ' bash "$SOURCE_DIR" "$PULL_DIR" >"$PULL_DIR/log" 2>&1 &
  echo $! >"$PULL_DIR/pid"
  log "OK: pull started pid=$(cat "$PULL_DIR/pid")"
  log "NEXT: call wait-pull after overlapping setup"
}

step_wait_pull() {
  log "PLAN: wait for background Coolify image pull"
  if [[ ! -d "$PULL_DIR" ]]; then
    log "OK: no background pull (caller will pull in the foreground)"
    return 0
  fi
  if [[ -f "$PULL_DIR/exit" ]]; then
    local rc
    rc="$(cat "$PULL_DIR/exit")"
    if [[ "$rc" != "0" ]]; then
      log "FAIL: background pull already exited $rc"
      tail -n 50 "$PULL_DIR/log" || true
      exit "$rc"
    fi
    log "OK: background pull already finished"
    return 0
  fi
  local i=0
  while [[ ! -f "$PULL_DIR/exit" ]]; do
    i=$((i + 1))
    if (( i % 5 == 0 )); then
      log "WAIT: docker compose pull still running (${i}s)"
    fi
    if (( i > 600 )); then
      log "FAIL: pull did not finish in 10 minutes"
      tail -n 80 "$PULL_DIR/log" || true
      exit 1
    fi
    sleep 1
  done
  local rc
  rc="$(cat "$PULL_DIR/exit")"
  if [[ "$rc" != "0" ]]; then
    log "FAIL: docker compose pull exited $rc"
    tail -n 80 "$PULL_DIR/log" || true
    exit "$rc"
  fi
  log "OK: background pull finished"
}

step_up() {
  log "PLAN: start Coolify stack from already-pulled images"
  cd "$SOURCE_DIR"
  log "DO: docker compose up -d --remove-orphans (180s cap)"
  # Unbounded compose up hung Setup Coolify for 45m on #795 while Acc
  # on the same run finished in 2m. Fail instead of eating the job timeout.
  run_limited 180 docker compose --env-file .env \
    -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.override.yml \
    up -d --remove-orphans
  log "OK: compose up returned"
}

step_wait_ready() {
  log "PLAN: poll Coolify /register until HTTP 200 or 302"
  local i
  for i in $(seq 1 "$READY_TRIES"); do
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/register 2>/dev/null | grep -qE "200|302"; then
      log "OK: Coolify is ready (attempt $i)"
      return 0
    fi
    if (( i % 5 == 0 )); then
      log "WAIT: /register not ready (attempt $i/$READY_TRIES)"
    fi
    sleep 2
  done
  log "FAIL: Coolify not ready after $READY_TRIES attempts"
  docker compose -f "$SOURCE_DIR/docker-compose.yml" logs --tail=50 || true
  exit 1
}

step_prepare_pull_bg() {
  step_prepare
  step_ssh
  step_pull_bg
}

case "$STEP" in
  prepare) step_prepare ;;
  ssh) step_ssh ;;
  pull) step_pull ;;
  pull-bg) step_pull_bg ;;
  wait-pull) step_wait_pull ;;
  up) step_up ;;
  wait-ready) step_wait_ready ;;
  prepare-pull-bg) step_prepare_pull_bg ;;
  *)
    echo "error: unknown step '$STEP'" >&2
    exit 2
    ;;
esac
