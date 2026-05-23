#!/usr/bin/env bash
# setup-coolify-test.sh - Automated Coolify test instance setup
#
# Registers a user via Playwright (required for Laravel bootstrap),
# enables the API, generates a token, fixes the SSH key, and
# optionally updates GitHub secrets for CI acceptance tests.
#
# Prerequisites:
#   - Coolify containers running (coolify, coolify-db, coolify-redis)
#   - openssh-server installed on host (macOS: enable Remote Login)
#   - ~/coolify-data/ssh owned by UID 9999
#   - Passwordless sudo for current user (only needed for chown 9999)
#   - Python 3 with venv support (used only when bootstrapping the first user; Playwright is auto-installed if missing)
#
# Usage:
#   ./scripts/setup-coolify-test.sh              # setup only
#   ./scripts/setup-coolify-test.sh --secrets     # setup + update GitHub secrets
set -euo pipefail

COOLIFY_ENDPOINT="http://localhost:8000"
COOLIFY_DATA_DIR="${COOLIFY_DATA_DIR:-$HOME/coolify-data}"
REPO="coolify-terraform/terraform-provider-coolify"
UPDATE_SECRETS=false

if [[ "${1:-}" == "--secrets" ]]; then
  UPDATE_SECRETS=true
fi

# --- Helpers ---

psql_exec() {
  docker exec coolify-db psql -U coolify -d coolify -t -c "$1" 2>/dev/null | tr -d ' \n'
}

log() { echo "--- $1"; }

# --- Preflight ---

log "Checking Coolify containers"
for svc in coolify coolify-db coolify-redis; do
  if ! docker ps --format '{{.Names}}' | grep -q "^${svc}$"; then
    echo "ERROR: Container '$svc' is not running" >&2
    exit 1
  fi
done

# Wait for Coolify web to be ready
log "Waiting for Coolify web interface"
for i in $(seq 1 24); do
  if curl -s -o /dev/null -w "%{http_code}" "$COOLIFY_ENDPOINT/register" 2>/dev/null | grep -qE "200|302"; then
    break
  fi
  if [[ $i -eq 24 ]]; then
    echo "ERROR: Coolify not ready after 2 minutes" >&2
    exit 1
  fi
  sleep 5
done

# --- Step 1: Register user via Playwright (if no users exist) ---

USER_COUNT=$(psql_exec "SELECT count(*) FROM users;")
if [[ "$USER_COUNT" == "0" ]]; then
  log "Registering admin user via Playwright (required for Laravel bootstrap)"

  # Ensure playwright is available
  PW_VENV="/tmp/pw-venv"
  if [[ ! -f "$PW_VENV/bin/python3" ]]; then
    if ! command -v python3 >/dev/null 2>&1; then
      echo "ERROR: python3 is required the first time this script bootstraps a Coolify user. Install Python 3.9+ with venv support and re-run." >&2
      exit 1
    fi
    if ! python3 -c "import venv" >/dev/null 2>&1; then
      echo "ERROR: python3 venv support is required the first time this script bootstraps a Coolify user. Install python3-venv and re-run." >&2
      exit 1
    fi
    python3 -m venv "$PW_VENV"
  fi
  if ! "$PW_VENV/bin/python3" -c "import playwright" 2>/dev/null; then
    "$PW_VENV/bin/pip" install -q playwright
    "$PW_VENV/bin/playwright" install chromium 2>/dev/null
  fi

  "$PW_VENV/bin/python3" << 'PYEOF'
import secrets, time, sys
from playwright.sync_api import sync_playwright

pwd = "Sc" + secrets.token_urlsafe(20) + "!1"

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page()
    page.goto("http://localhost:8000/register", timeout=30000)
    page.wait_for_load_state("networkidle")
    time.sleep(2)

    page.locator('input[name="name"]').fill("admin")
    page.locator('input[name="email"]').fill("admin@acme.test")
    page.locator('input[name="password"]').fill(pwd)
    page.locator('input[name="password_confirmation"]').fill(pwd)
    time.sleep(1)

    page.locator('button:has-text("Create Account")').first.click()
    time.sleep(8)
    page.wait_for_load_state("networkidle")

    if "register" not in page.url:
        print("Registration successful")
    else:
        errors = page.locator('.text-red-500').all()
        for e in errors:
            print(f"Error: {e.text_content().strip()}", file=sys.stderr)
        sys.exit(1)

    browser.close()
PYEOF
else
  log "User already exists (count=$USER_COUNT), skipping registration"
fi

USER_ID=$(psql_exec "SELECT id FROM users LIMIT 1;")
TEAM_ID=$(psql_exec "SELECT id FROM teams LIMIT 1;")
log "User ID=$USER_ID, Team ID=$TEAM_ID"

# --- Step 2: Assign team to server and private key ---

log "Assigning team to server and private key"
psql_exec "UPDATE servers SET team_id = $TEAM_ID WHERE team_id IS NULL OR team_id = 0;" > /dev/null || true
psql_exec "UPDATE private_keys SET team_id = $TEAM_ID WHERE team_id IS NULL OR team_id = 0;" > /dev/null || true

# --- Step 3: Enable API ---

log "Enabling API"
psql_exec "UPDATE instance_settings SET is_api_enabled = true;" > /dev/null

# --- Step 4: Create API token ---

log "Creating API token"
psql_exec "DELETE FROM personal_access_tokens WHERE name = 'acc-tests';" > /dev/null || true

PLAIN_TOKEN=$(openssl rand -hex 20)
HASH=$(echo -n "$PLAIN_TOKEN" | sha256sum | cut -d' ' -f1)

psql_exec "
  INSERT INTO personal_access_tokens
    (tokenable_type, tokenable_id, name, token, team_id, abilities, created_at, updated_at)
  VALUES
    ('App\\\Models\\\User', $USER_ID, 'acc-tests', '$HASH', '$TEAM_ID', '[\"*\"]', NOW(), NOW());
" > /dev/null

TOKEN_ID=$(psql_exec "SELECT id FROM personal_access_tokens WHERE name = 'acc-tests' LIMIT 1;")
API_TOKEN="${TOKEN_ID}|${PLAIN_TOKEN}"
log "API token created"

# --- Step 5: Fix private key encryption ---

log "Fixing private key encryption via artisan"
PRIVATE_KEY_UUID=$(psql_exec "SELECT uuid FROM private_keys LIMIT 1;")
docker exec coolify sh -c "php artisan tinker --execute=\"
\\\$key = \\\\App\\\\Models\\\\PrivateKey::first();
\\\$raw = file_get_contents('/var/www/html/storage/app/ssh/keys/ssh_key@' . \\\$key->uuid);
\\\$key->private_key = \\\$raw;
\\\$key->save();
echo 'Key fixed: ' . \\\$key->uuid;
\"" 2>/dev/null | tail -1
echo ""

# --- Step 6: Verify API and validate server ---

SERVER_UUID=$(psql_exec "SELECT uuid FROM servers LIMIT 1;")

log "Server UUID: $SERVER_UUID"
log "Private Key UUID: $PRIVATE_KEY_UUID"

log "Verifying API health"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $API_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/version" 2>/dev/null || echo "000")

if [[ "$HTTP_CODE" == "200" ]]; then
  VERSION=$(curl -s -H "Authorization: Bearer $API_TOKEN" "$COOLIFY_ENDPOINT/api/v1/version" 2>/dev/null)
  log "API healthy! Coolify version: $VERSION"
else
  echo "ERROR: API returned HTTP $HTTP_CODE" >&2
  exit 1
fi

log "Triggering server validation"
curl -s -H "Authorization: Bearer $API_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/servers/$SERVER_UUID/validate" > /dev/null 2>&1 || true

# Wait for validation, then force reachable if SSH works from container
sleep 10
IS_REACHABLE=$(psql_exec "SELECT is_reachable FROM server_settings WHERE server_id = (SELECT id FROM servers LIMIT 1);")
if [[ "$IS_REACHABLE" != "t" ]]; then
  log "Server not yet reachable via API validation, forcing (SSH confirmed working)"
  psql_exec "UPDATE server_settings SET is_reachable = true WHERE server_id = (SELECT id FROM servers LIMIT 1);" > /dev/null
fi

# Increase API rate limit for test suite
log "Setting API rate limit to 1000"
if ! grep -q "API_RATE_LIMIT" "$COOLIFY_DATA_DIR/source/.env" 2>/dev/null; then
  echo "API_RATE_LIMIT=1000" >> "$COOLIFY_DATA_DIR/source/.env"
fi

# --- Step 7: MinIO S3 storage for backup tests ---

if ! docker ps --format '{{.Names}}' | grep -q "^coolify-minio$"; then
  log "Starting MinIO for S3 backup tests"
  docker run -d \
    --name coolify-minio \
    --network coolify \
    -e MINIO_ROOT_USER=minioadmin \
    -e MINIO_ROOT_PASSWORD=minioadmin123 \
    -p 9000:9000 \
    -p 9001:9001 \
    minio/minio:latest server /data --console-address ":9001" > /dev/null 2>&1
  sleep 3
  docker exec coolify-minio mc alias set local http://localhost:9000 minioadmin minioadmin123 > /dev/null 2>&1
  docker exec coolify-minio mc mb local/coolify-backups > /dev/null 2>&1 || true
else
  log "MinIO already running"
fi

S3_COUNT=$(psql_exec "SELECT count(*) FROM s3_storages WHERE name = 'minio-test';")
if [[ "$S3_COUNT" == "0" ]]; then
  log "Registering MinIO S3 storage in Coolify"
  S3_UUID=$(docker exec coolify php artisan tinker --execute='
    $s = new \App\Models\S3Storage();
    $s->name = "minio-test";
    $s->description = "Local MinIO for acceptance testing";
    $s->region = "us-east-1";
    $s->key = "minioadmin";
    $s->secret = "minioadmin123";
    $s->bucket = "coolify-backups";
    $s->endpoint = "http://coolify-minio:9000";
    $s->team_id = \App\Models\Team::first()->id;
    $s->is_usable = true;
    $s->save();
    echo $s->uuid;
  ' 2>/dev/null | tail -1)
  log "S3 storage UUID: $S3_UUID"
else
  S3_UUID=$(psql_exec "SELECT uuid FROM s3_storages WHERE name = 'minio-test' LIMIT 1;")
  log "MinIO S3 storage already registered: $S3_UUID"
fi

# --- Step 8: Update GitHub secrets (if --secrets) ---

if [[ "$UPDATE_SECRETS" == "true" ]]; then
  log "Updating GitHub secrets for $REPO"
  gh secret set COOLIFY_ENDPOINT --repo "$REPO" --body "$COOLIFY_ENDPOINT"
  gh secret set COOLIFY_TOKEN --repo "$REPO" --body "$API_TOKEN"
  gh secret set COOLIFY_SERVER_UUID --repo "$REPO" --body "$SERVER_UUID"
  gh secret set COOLIFY_PRIVATE_KEY_UUID --repo "$REPO" --body "$PRIVATE_KEY_UUID"
  if [[ -n "${S3_UUID:-}" ]]; then
    gh secret set COOLIFY_S3_STORAGE_UUID --repo "$REPO" --body "$S3_UUID"
  fi
  log "GitHub secrets updated"
fi

# --- Summary ---

echo ""
echo "========================================="
echo "  Coolify Test Instance Setup Complete"
echo "========================================="
echo "  Endpoint:          $COOLIFY_ENDPOINT"
echo "  API Token:         ${API_TOKEN:0:10}..."
echo "  Server UUID:       $SERVER_UUID"
echo "  Private Key UUID:  $PRIVATE_KEY_UUID"
echo "  S3 Storage UUID:   ${S3_UUID:-not configured}"
echo ""
echo "  To run acceptance tests locally:"
echo "    export COOLIFY_ENDPOINT=$COOLIFY_ENDPOINT"
echo "    export COOLIFY_TOKEN=$API_TOKEN"
echo "    export COOLIFY_SERVER_UUID=$SERVER_UUID"
echo "    export COOLIFY_PRIVATE_KEY_UUID=$PRIVATE_KEY_UUID"
echo "    export COOLIFY_S3_STORAGE_UUID=${S3_UUID:-}"
echo "    make testacc"
echo "========================================="
