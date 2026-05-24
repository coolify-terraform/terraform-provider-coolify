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
if [[ -z "$USER_ID" || -z "$TEAM_ID" ]]; then
  echo "ERROR: Could not find user or team in database (user=$USER_ID, team=$TEAM_ID)" >&2
  echo "DEBUG: users table:" >&2
  docker exec coolify-db psql -U coolify -d coolify -c "SELECT id, name, email FROM users;" 2>/dev/null >&2
  echo "DEBUG: teams table:" >&2
  docker exec coolify-db psql -U coolify -d coolify -c "SELECT id, name FROM teams;" 2>/dev/null >&2
  exit 1
fi
log "User ID=$USER_ID, Team ID=$TEAM_ID"

# --- Step 2: Assign team to server and private key ---

log "Assigning team to server and private key"
psql_exec "UPDATE servers SET team_id = $TEAM_ID WHERE team_id IS NULL OR team_id = 0;" > /dev/null || true
psql_exec "UPDATE private_keys SET team_id = $TEAM_ID WHERE team_id IS NULL OR team_id = 0;" > /dev/null || true

# --- Step 3: Enable API and clear cache ---

log "Enabling API via artisan"
docker exec coolify php artisan tinker --execute='
  $s = \App\Models\InstanceSettings::first();
  $s->is_api_enabled = true;
  $s->save();
  echo "API enabled: " . ($s->is_api_enabled ? "true" : "false");
' 2>/dev/null | tail -1
docker exec coolify php artisan config:clear 2>/dev/null || true

# --- Step 4: Create API token ---

log "Creating API token"

# Use artisan tinker to create token through Sanctum (proper hashing).
# team_id is set explicitly because currentTeam() returns null on fresh installs.
TINKER_OUTPUT=$(docker exec coolify php artisan tinker --execute='
  $user = \App\Models\User::first();
  if (!$user) { echo "ERROR: no user"; exit(1); }
  $team = \App\Models\Team::first();
  if (!$team) { echo "ERROR: no team"; exit(1); }
  $user->tokens()->where("name", "acc-tests")->delete();
  $token = $user->createToken("acc-tests", ["*"]);
  $pat = $token->accessToken;
  $pat->team_id = $team->id;
  $pat->save();
  echo $token->plainTextToken;
' 2>&1)

API_TOKEN=$(echo "$TINKER_OUTPUT" | grep -oE '^[0-9]+\|[A-Za-z0-9]+' | tail -1 || true)

if [[ -z "$API_TOKEN" ]]; then
  echo "ERROR: Failed to create API token. Artisan output:" >&2
  echo "$TINKER_OUTPUT" >&2
  exit 1
fi
log "API token: ${API_TOKEN:0:8}..."

# --- Step 5: Fix private key encryption ---

log "Fixing private key encryption via artisan"
PRIVATE_KEY_UUID=$(psql_exec "SELECT uuid FROM private_keys LIMIT 1;")
KEY_FIX_OUTPUT=$(docker exec coolify php artisan tinker --execute='
  $key = \App\Models\PrivateKey::first();
  if (!$key) { echo "ERROR: No private key found"; exit(1); }
  $path = "/var/www/html/storage/app/ssh/keys/ssh_key@" . $key->uuid;
  if (!file_exists($path)) { echo "ERROR: Key file not found: " . $path; exit(1); }
  $raw = file_get_contents($path);
  $key->private_key = $raw;
  $key->save();
  echo "Key fixed: " . $key->uuid;
' 2>&1 | tail -1)
log "$KEY_FIX_OUTPUT"

# Allow Coolify to settle after database changes
log "Waiting 10s for Coolify to pick up settings changes"
sleep 10

# --- Step 6: Verify API and validate server ---

SERVER_UUID=$(psql_exec "SELECT uuid FROM servers LIMIT 1;")

log "Server UUID: $SERVER_UUID"
log "Private Key UUID: $PRIVATE_KEY_UUID"

log "Verifying API health (with retries)"

# Quick no-auth diagnostic to distinguish "API is down" from "auth is broken"
NO_AUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Accept: application/json" \
  "$COOLIFY_ENDPOINT/api/v1/version" 2>/dev/null || echo "000")
log "No-auth check: HTTP $NO_AUTH_CODE (401=API works but needs auth, 500=app error)"

API_OK=false
for attempt in $(seq 1 12); do
  RESP_FILE=$(mktemp)
  HTTP_CODE=$(curl -s -o "$RESP_FILE" -w "%{http_code}" \
    -H "Authorization: Bearer $API_TOKEN" \
    -H "Accept: application/json" \
    "$COOLIFY_ENDPOINT/api/v1/version" 2>/dev/null || echo "000")

  if [[ "$HTTP_CODE" == "200" ]]; then
    VERSION=$(cat "$RESP_FILE" | tr -d '"')
    log "API healthy! Coolify version: $VERSION (attempt $attempt)"
    API_OK=true
    rm -f "$RESP_FILE"
    break
  fi

  log "API returned HTTP $HTTP_CODE (attempt $attempt/12), retrying in 5s..."
  if [[ "$HTTP_CODE" == "500" ]]; then
    echo "  Response body: $(cat "$RESP_FILE")" >&2
  fi
  rm -f "$RESP_FILE"
  sleep 5
done

if [[ "$API_OK" != "true" ]]; then
  echo "ERROR: API not healthy after 12 attempts" >&2
  echo "DEBUG: Coolify container logs (last 30 lines):" >&2
  docker logs coolify --tail 30 2>&1 >&2 || true
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

# --- Step 9: Write env file for CI (if requested) ---

ENV_FILE="${COOLIFY_ENV_FILE:-}"
if [[ -n "$ENV_FILE" ]]; then
  log "Writing env file to $ENV_FILE"
  cat > "$ENV_FILE" <<EOF
COOLIFY_ENDPOINT=$COOLIFY_ENDPOINT
COOLIFY_TOKEN=$API_TOKEN
COOLIFY_SERVER_UUID=$SERVER_UUID
COOLIFY_PRIVATE_KEY_UUID=$PRIVATE_KEY_UUID
COOLIFY_S3_STORAGE_UUID=${S3_UUID:-}
EOF
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
