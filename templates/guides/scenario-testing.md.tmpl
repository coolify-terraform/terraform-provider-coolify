---
page_title: "Running Scenario Tests"
subcategory: "Development"
description: |-
  How to run the ACME Corp scenario examples against a real Coolify instance.
---

# Running Scenario Tests

The `examples/scenarios/` directory contains real-world deployment scenarios
based on a fictional ACME Corp company. Each scenario has working Terraform
code and a `.tftest.hcl` file that validates the infrastructure against a
real Coolify instance.

-> **This is different from `make testacc`.** The Go acceptance tests
(`make testacc`) test individual resources via the Go test framework.
Scenario tests use `terraform test` to validate multi-resource composition
from the same `.tf` files that users copy. Both test against a real
Coolify instance.

## What "pass" means

A passing scenario test means:
- All Terraform resources were created successfully on real Coolify
- The API returned valid UUIDs for every resource
- Resource references (e.g. app -> database via env var) resolved correctly
- `terraform test` auto-destroyed all resources after verification

It does NOT mean the applications are building, running, or serving
traffic. Coolify queues builds asynchronously; the test only validates
that the Terraform provider creates and reads resources correctly.

## Prerequisites

- Docker and Docker Compose
- Go 1.26+ (to build the provider)
- Terraform 1.12+ (for `terraform test`)
- `sudo` access (one-time setup for directories, SSH, and sudo config)
- OpenSSH server (Coolify SSHs into the target server to manage Docker)
- Port 8000 available (or change `APP_PORT` in `.env`)

-> **Do not use the `docker-compose.yml` in the repo root.** That file
is a minimal single-container setup for reference only. Coolify requires
4 services (app + PostgreSQL + Redis + Soketi). This guide uses the
official multi-service installation.

## Step 1: Host Prerequisites (one-time)

Coolify is a PaaS that manages Docker containers by SSHing into target
servers. Even when the "server" is localhost, Coolify connects via SSH.
These prerequisites make that work:

```bash
# Install OpenSSH server (Coolify needs to SSH into the host)
sudo apt-get install -y openssh-server
sudo systemctl enable ssh && sudo systemctl start ssh

# Allow passwordless sudo (Coolify runs sudo commands on the target)
echo "$(whoami) ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/coolify-test
```

## Step 2: Set Up a Local Coolify Instance

Coolify requires PostgreSQL, Redis, and Soketi. Use the official
multi-service installation (NOT the single-container docker-compose.yml
in this repo, which doesn't work):

```bash
# Create directories
sudo mkdir -p /data/coolify/{source,ssh/{keys,mux},applications,databases,backups,services,proxy,webhooks-during-maintenance}
sudo mkdir -p /data/coolify/proxy/dynamic
sudo chown -R $USER:$USER /data/coolify

# Download official compose files
cd /data/coolify/source
curl -fsSL https://cdn.coollabs.io/coolify/docker-compose.yml -o docker-compose.yml
curl -fsSL https://cdn.coollabs.io/coolify/docker-compose.prod.yml -o docker-compose.prod.yml
curl -fsSL https://cdn.coollabs.io/coolify/.env.production -o .env

# Generate secrets
sed -i "s|APP_ID=.*|APP_ID=$(openssl rand -hex 16)|g" .env
sed -i "s|APP_KEY=.*|APP_KEY=base64:$(openssl rand -base64 32)|g" .env
sed -i "s|DB_PASSWORD=.*|DB_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
sed -i "s|REDIS_PASSWORD=.*|REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
sed -i "s|PUSHER_APP_ID=.*|PUSHER_APP_ID=$(openssl rand -hex 32)|g" .env
sed -i "s|PUSHER_APP_KEY=.*|PUSHER_APP_KEY=$(openssl rand -hex 32)|g" .env
sed -i "s|PUSHER_APP_SECRET=.*|PUSHER_APP_SECRET=$(openssl rand -hex 32)|g" .env

# Generate SSH keys for localhost server management
ssh-keygen -t ed25519 -f /data/coolify/ssh/keys/id.root@host.docker.internal -N "" -q
mkdir -p ~/.ssh
cat /data/coolify/ssh/keys/id.root@host.docker.internal.pub >> ~/.ssh/authorized_keys
chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys

# Set SSH directory ownership for the Coolify container user (UID 9999)
sudo chown -R 9999 /data/coolify/ssh
sudo chmod -R 700 /data/coolify/ssh

# Create Docker network and start
docker network create --attachable coolify
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml \
  up -d --pull always --remove-orphans --force-recreate
```

If port 8000 is already in use, edit `/data/coolify/source/.env` and set
`APP_PORT=9000` (or any free port) before starting. Use that port in all
subsequent commands and `TF_VAR_coolify_endpoint`.

Wait 30-60 seconds for Coolify to start, then verify:

```bash
curl -s http://localhost:8000/api/health
```

-> **Why SSH?** Coolify deploys applications by SSHing into target servers
and running Docker commands. Without a reachable SSH server, Terraform
fails with `Application created but not persisted` after the API returns a
UUID and the provider cannot read the application back.

## Step 3: Create an Admin Account and Enable the API

### Option A: Browser (if you have a desktop)

1. Open [http://localhost:8000](http://localhost:8000) and register an
   admin account. Use a unique password (Coolify rejects passwords found
   in the [Have I Been Pwned](https://haveibeenpwned.com/) database).
2. After registration, Coolify shows an **onboarding wizard**. You can
   complete it or skip it, but you MUST dismiss it before accessing
   Settings (the Settings page returns 404 until onboarding is done).
3. You may also see a **sponsor/donation popup** after login. Dismiss it
   with "Accept and Close" or "Maybe next time".
4. Go to **Settings** and enable the **API**.
5. Go to **Security > API Tokens**, create a token, and copy it.

### Option B: Database only (headless servers, CI)

If you don't have a browser (headless server, CI runner), do everything
via the database. You still need to register through the UI for the
initial admin account, but you can use a tool like `curl` with CSRF
tokens or skip registration entirely by inserting the user directly.

The simplest headless approach is to enable the API and create a token
via the database after registering through a forwarded port
(`ssh -L 8000:localhost:8000 your-server`):

```bash
# Enable API
docker exec coolify-db psql -U coolify -d coolify \
  -c "UPDATE instance_settings SET is_api_enabled = true;"

# Create token
PLAIN_TOKEN=$(openssl rand -hex 20)
HASH=$(echo -n "$PLAIN_TOKEN" | sha256sum | cut -d' ' -f1)
USER_ID=$(docker exec coolify-db psql -U coolify -d coolify -t \
  -c "SELECT id FROM users LIMIT 1;" | tr -d ' ')
TEAM_ID=$(docker exec coolify-db psql -U coolify -d coolify -t \
  -c "SELECT id FROM teams LIMIT 1;" | tr -d ' ')

docker exec coolify-db psql -U coolify -d coolify -c "
INSERT INTO personal_access_tokens
  (tokenable_type, tokenable_id, name, token, team_id, abilities, created_at, updated_at)
VALUES
  ('App\\Models\\User', $USER_ID, 'scenario-tests', '$HASH', '$TEAM_ID', '[\"*\"]', NOW(), NOW())
RETURNING id;"

TOKEN_ID=$(docker exec coolify-db psql -U coolify -d coolify -t \
  -c "SELECT id FROM personal_access_tokens WHERE name='scenario-tests' LIMIT 1;" | tr -d ' ')

export COOLIFY_TOKEN="${TOKEN_ID}|${PLAIN_TOKEN}"
echo "Token: $COOLIFY_TOKEN"
```

Verify it works:

```bash
curl -s http://localhost:8000/api/v1/version -H "Authorization: Bearer $COOLIFY_TOKEN"
# Should print: 4.0.0
```

## Step 4: Register a Server

```bash
# Upload the SSH key
KEY_UUID=$(curl -s -X POST http://localhost:8000/api/v1/security/keys \
  -H "Authorization: Bearer $COOLIFY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$(python3 -c "import json; print(json.dumps({
    'name': 'localhost-key',
    'private_key': open('/data/coolify/ssh/keys/id.root@host.docker.internal').read()
  }))")" | python3 -c "import sys,json;print(json.load(sys.stdin)['uuid'])")

# Fix SSH key encryption (API-created keys may not encrypt properly)
docker exec coolify sh -c 'php artisan tinker --execute="
\$key = \App\Models\PrivateKey::first();
\$raw = file_get_contents(\"/var/www/html/storage/app/ssh/keys/id.root@host.docker.internal\");
\$key->private_key = \$raw;
\$key->save();
echo \"Key fixed: \" . \$key->uuid;
"'

# Register the server
export COOLIFY_SERVER_UUID=$(curl -s -X POST http://localhost:8000/api/v1/servers \
  -H "Authorization: Bearer $COOLIFY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"localhost\",
    \"ip\": \"host.docker.internal\",
    \"port\": 22,
    \"user\": \"$(whoami)\",
    \"private_key_uuid\": \"$KEY_UUID\"
  }" | python3 -c "import sys,json;print(json.load(sys.stdin)['uuid'])")

echo "Server UUID: $COOLIFY_SERVER_UUID"

# Validate the server (triggers SSH connectivity check)
curl -s "http://localhost:8000/api/v1/servers/$COOLIFY_SERVER_UUID/validate" \
  -H "Authorization: Bearer $COOLIFY_TOKEN"

# Wait for validation to complete (usually 10-20 seconds)
for i in $(seq 1 12); do
  sleep 10
  STATUS=$(curl -s "http://localhost:8000/api/v1/servers/$COOLIFY_SERVER_UUID" \
    -H "Authorization: Bearer $COOLIFY_TOKEN" | \
    python3 -c "import sys,json; s=json.load(sys.stdin).get('settings',{}); \
    print(f'{s.get(\"is_reachable\",False)} {s.get(\"is_usable\",False)}')")
  echo "($i/12) reachable/usable: $STATUS"
  echo "$STATUS" | grep -q "True True" && echo "Server ready!" && break
done
```

-> **If validation fails**, check: (1) `sshd` is running on the host,
(2) `/data/coolify/ssh` is owned by UID 9999, (3) passwordless sudo
is configured. Run `docker logs coolify 2>&1 | grep -i validate` for
the actual error.

## Step 5: Build and Install the Provider

```bash
cd /path/to/terraform-provider-coolify
go build -o terraform-provider-coolify .

PLUGIN_DIR="$HOME/.terraform.d/plugins/SebTardifLabs/coolify/0.0.0-dev/linux_amd64"
mkdir -p "$PLUGIN_DIR"
cp terraform-provider-coolify "$PLUGIN_DIR/"

cat > "$HOME/.terraformrc" << EOF
provider_installation {
  dev_overrides {
    "SebTardifLabs/coolify" = "$PLUGIN_DIR"
  }
  direct {}
}
EOF
```

-> **Warning:** While `~/.terraformrc` exists with `dev_overrides`, ALL
`terraform` commands on this machine use the dev provider binary instead
of the published one. This affects other Terraform projects too. Delete
`~/.terraformrc` when you're done testing (see Cleanup).

## Step 6: Run the Scenarios

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="$COOLIFY_TOKEN"
export TF_VAR_server_uuid="$COOLIFY_SERVER_UUID"

# Run a single scenario
cd examples/scenarios/acme-website
terraform test -verbose

# Run all scenarios
for dir in examples/scenarios/acme-*/; do
  echo "=== Testing $(basename "$dir") ==="
  cd "$dir"
  if [ -d "modules" ]; then
    # terraform init still resolves providers with dev_overrides, so fetch only local modules here.
    terraform get
  fi
  terraform test -verbose
  cd -
done
```

## Coming Back Later

Coolify data persists in Docker volumes (`coolify-db`, `coolify-redis`).
After stopping and restarting, you don't need to re-register or recreate
the token. Just start Coolify and re-export your variables:

```bash
# Start Coolify
cd /data/coolify/source
docker network create --attachable coolify 2>/dev/null
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml up -d

# Re-export your saved token and server UUID
export COOLIFY_TOKEN="<your-token-from-step-3>"
export COOLIFY_SERVER_UUID="<your-uuid-from-step-4>"
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="$COOLIFY_TOKEN"
export TF_VAR_server_uuid="$COOLIFY_SERVER_UUID"
```

-> **Save your token and server UUID** somewhere after the initial setup.
You'll need them every time you restart.

If you run `docker compose down -v` (with `-v`), all data is deleted
and you must redo Steps 3-4.

## Cleanup

```bash
# Remove provider dev setup
rm -f terraform-provider-coolify ~/.terraformrc
rm -rf ~/.terraform.d/plugins/SebTardifLabs

# Stop Coolify (data persists)
cd /data/coolify/source
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml down

# Full teardown (removes ALL data)
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml down -v
docker network rm coolify 2>/dev/null
sudo rm -rf /data/coolify
```

## Troubleshooting

### "Validation failed" (422) on resource creation

Coolify's API rejects unexpected fields. If you see this on environment
variables, check that the provider version matches the API version. Run
`go build` to rebuild the provider after any code changes.

### Server shows `is_reachable=False`

Check in order:
1. **SSH server running?** `sudo systemctl status ssh`
2. **SSH key readable by Coolify?** `docker exec coolify ls -la /var/www/html/storage/app/ssh/keys/`
   (files must be owned by `www-data` / UID 9999)
3. **Passwordless sudo?** `sudo -n true && echo "OK"` (should print OK with no prompt)
4. **Actual error?** `docker logs coolify 2>&1 | grep -i "validate\|FAIL" | tail -5`

### "Project has resources, so it cannot be deleted"

Coolify deletes applications asynchronously. When `terraform destroy`
tries to delete the project immediately after the app, the app hasn't
been fully removed yet. Wait a few seconds and delete the project
manually: `curl -X DELETE ".../projects/{uuid}"`

### Applications created but not persisted (404 on read-back)

The server is not SSH-reachable. Coolify may return a UUID from Create,
but the provider then fails with `Application created but not persisted`
when the read-back 404s. Fix the server validation (see Step 4).

## Available Scenarios

| Scenario | Level | What It Tests |
|---|---|---|
| `acme-website` | Beginner | Project + PostgreSQL + Node.js app + env vars + optional S3 backups |
| `acme-api` | Intermediate | Dockerfile app + Docker image worker + PostgreSQL + Redis + scheduled task + backup |
| `acme-databases` | Beginner | All 8 database engines (PostgreSQL, MySQL, MariaDB, Redis, MongoDB, ClickHouse, KeyDB, Dragonfly) |
| `acme-docker` | Intermediate | Docker image tag normalization, scheduled tasks, storage |
| `acme-integrations` | Intermediate | Managed services from the Coolify catalog |
| `acme-multi-env` | Advanced | Terraform module composition for dev + staging environments |
| `acme-platform` | Intermediate | Private keys, environments, storage, data sources |
| `acme-backups` | Beginner | Full backup lifecycle: schedule, trigger, monitor execution status |
