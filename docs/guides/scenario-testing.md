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

## Prerequisites

- Docker and Docker Compose
- Go 1.26+ (to build the provider)
- Terraform 1.12+ (for `terraform test`)
- A machine where you can run `sudo` (one-time directory setup)

## Step 1: Set Up a Local Coolify Instance

Coolify requires PostgreSQL, Redis, and Soketi. Use the official multi-service
installation:

```bash
# Create directories (one-time, needs sudo)
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

# Create Docker network and start
docker network create --attachable coolify
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml \
  up -d --pull always --remove-orphans --force-recreate
```

Wait 30-60 seconds for Coolify to start, then verify:

```bash
curl -s http://localhost:8000/api/health
```

## Step 2: Create an Admin Account

Open [http://localhost:8000](http://localhost:8000) in your browser and
create the first admin account. Use a strong, unique password (Coolify
rejects passwords found in data breaches).

## Step 3: Enable the API and Create a Token

In the Coolify dashboard:

1. Go to **Settings** and enable the API
2. Go to **Security > API Tokens** and create a new token
3. Copy the token (it starts with a number followed by `|`)

Alternatively, enable the API and create a token via the database:

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
```

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
  # Modules need terraform init even with dev_overrides
  [ -d "modules" ] && terraform init
  terraform test -verbose
  cd -
done
```

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

## Available Scenarios

| Scenario | Level | What It Tests |
|---|---|---|
| `acme-website` | Beginner | Project + PostgreSQL + Node.js app + env vars + optional S3 backups |
| `acme-api` | Intermediate | Dockerfile app + Docker image worker + PostgreSQL + Redis + scheduled task + backup |
| `acme-multi-env` | Advanced | Terraform module composition for dev + staging environments |
