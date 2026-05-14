# Testing Guide

This provider has two kinds of tests: **unit tests** (against mock HTTP
servers) and **acceptance tests** (against a real Coolify instance).

## Unit Tests

Unit tests use `httptest.NewServer` to simulate the Coolify API. They
validate code logic (schema mapping, state handling, error paths) without
requiring any external infrastructure.

```bash
make test
```

All unit tests run with `-race` detection and must pass in CI.

### Writing a unit test

1. Create `resource_test.go` in the resource's package
2. Build a mock HTTP server handling the CRUD endpoints
3. Wrap with `acctest.WithVersionEndpoint()` (provider health check)
4. Use `resource.UnitTest()` and `acctest.TestProtoV6ProviderFactories()`
5. Use `acctest.TestResourceConfig()` for HCL config generation

Every resource test should include these scenarios:
- **Create** with attribute verification
- **Idempotency** (`PlanOnly: true, ExpectNonEmptyPlan: false`)
- **Update** (change attributes, verify plan applied)
- **Import** (import by ID, verify state matches)
- **Disappears** (simulate out-of-band deletion)
- **CheckDestroy** (verify cleanup after `terraform destroy`)

## Acceptance Tests

Acceptance tests run against a real Coolify instance. They exercise the
full Terraform lifecycle: `plan` -> `apply` -> `read` -> `update` ->
`import` -> `destroy`.

### Prerequisites

Use the official multi-service installation to set up a local Coolify instance:

```bash
# 1. Create directories (needs sudo)
sudo mkdir -p /data/coolify/{source,ssh/{keys,mux},applications,databases,backups,services,proxy,webhooks-during-maintenance}
sudo mkdir -p /data/coolify/proxy/dynamic
sudo chown -R $USER:$USER /data/coolify

# 2. Download official compose files
cd /data/coolify/source
curl -fsSL https://cdn.coollabs.io/coolify/docker-compose.yml -o docker-compose.yml
curl -fsSL https://cdn.coollabs.io/coolify/docker-compose.prod.yml -o docker-compose.prod.yml
curl -fsSL https://cdn.coollabs.io/coolify/.env.production -o .env

# 3. Generate secrets
sed -i "s|APP_ID=.*|APP_ID=$(openssl rand -hex 16)|g" .env
sed -i "s|APP_KEY=.*|APP_KEY=base64:$(openssl rand -base64 32)|g" .env
sed -i "s|DB_PASSWORD=.*|DB_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
sed -i "s|REDIS_PASSWORD=.*|REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=')|g" .env
sed -i "s|PUSHER_APP_ID=.*|PUSHER_APP_ID=$(openssl rand -hex 32)|g" .env
sed -i "s|PUSHER_APP_KEY=.*|PUSHER_APP_KEY=$(openssl rand -hex 32)|g" .env
sed -i "s|PUSHER_APP_SECRET=.*|PUSHER_APP_SECRET=$(openssl rand -hex 32)|g" .env

# 4. Generate SSH keys for localhost server
ssh-keygen -t ed25519 -f /data/coolify/ssh/keys/id.root@host.docker.internal -N "" -q
mkdir -p ~/.ssh
cat /data/coolify/ssh/keys/id.root@host.docker.internal.pub >> ~/.ssh/authorized_keys
chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys

# 5. Create Docker network and start
docker network create --attachable coolify
cd /data/coolify/source
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml \
  up -d --pull always --remove-orphans --force-recreate
```

Wait 30-60 seconds for Coolify to start, then complete registration at
http://localhost:8000/register.

**Important**: The API is disabled by default on fresh installs. Enable it
via the Settings page in the UI, or directly in the database:

```bash
docker exec coolify-db psql -U coolify -d coolify \
  -c "UPDATE instance_settings SET is_api_enabled = true;"
```

Then create an API token from **Settings > API Tokens** in the UI, or via
the database (see the `coolify-test-instance` skill for the database method).

```bash
# Set environment variables
export COOLIFY_ENDPOINT="http://localhost:8000"
export COOLIFY_TOKEN="<your-api-token>"

# Optional: set a specific server UUID (otherwise auto-discovered)
export COOLIFY_SERVER_UUID="<server-uuid>"
```

#### Server validation for application tests

Application-related tests require a validated server with SSH access.
Without this, application creation can fail with `Application created but
not persisted` when the provider cannot read the app back. See the
[coolify-test-instance skill](../../.grok/skills/coolify-test-instance/SKILL.md)
for the full SSH and server setup procedure.

### Running acceptance tests

**Important**: Running all tests in parallel can overwhelm the Coolify API
and cause false timeout failures. Use `-p 1` to run packages sequentially:

```bash
# Run all acceptance tests (sequential packages, avoids API overload)
TF_ACC=1 go test -race -v -cover -count=1 -timeout=120m -p 1 -run 'TestAcc' ./...

# Run a specific test
TF_ACC=1 go test -v -count=1 -timeout=30m \
  -run TestAccProjectResource_CRUD ./internal/service/project/

# Run all acceptance tests for a package
TF_ACC=1 go test -v -count=1 -timeout=30m \
  -run TestAcc ./internal/service/application/
```

### Writing an acceptance test

#### File naming

Acceptance tests go in separate files from unit tests:
- Resources: `resource_acc_test.go`
- Data sources: `data_source_acc_test.go`

#### Test function pattern

```go
func TestAccMyResource_CRUD(t *testing.T) {
    t.Parallel()
    acctest.AccTestSkipIfNoTFAcc(t)  // skip if TF_ACC not set
    acctest.TestAccPreCheck(t)        // verify ENDPOINT + TOKEN
    name := acctest.RandomWithPrefix("tf-acc-myres")

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
        Steps: []resource.TestStep{
            // Create
            {
                Config: myResourceConfig(name, ""),
                Check:  resource.TestCheckResourceAttrSet("coolify_my_resource.test", "uuid"),
            },
            // Update
            {
                Config: myResourceConfig(name, `description = "updated"`),
                Check:  resource.TestCheckResourceAttr("coolify_my_resource.test", "description", "updated"),
            },
            // Import
            {
                ResourceName:                         "coolify_my_resource.test",
                ImportState:                          true,
                ImportStateVerify:                    true,
                ImportStateVerifyIdentifierAttribute: "uuid",
            },
        },
    })
}
```

#### Key differences from unit tests

| Aspect | Unit Test | Acceptance Test |
|--------|-----------|-----------------|
| Function prefix | `TestMyResource_Create` | `TestAccMyResource_CRUD` |
| Test function | `resource.UnitTest()` | `resource.Test()` |
| Provider config | `acctest.ProviderBlockForURL(srv.URL)` | `acctest.ConfigProviderBlock()` |
| HTTP server | `httptest.NewServer(...)` | Real Coolify instance |
| Server UUID | Hardcoded in mock | `acctest.AccTestServerUUID(t)` |
| Guard | None | `AccTestSkipIfNoTFAcc` + `TestAccPreCheck` |

#### Resources that need a server

Many resources (applications, databases, services) require a server UUID.
Use `acctest.AccTestServerUUID(t)` which:
1. Checks `COOLIFY_SERVER_UUID` env var first
2. Falls back to querying the API for the first available server
3. Skips the test if no server is available

#### Dependency chains

Dependent resources should be created in a single Terraform config using
resource references (not hardcoded UUIDs):

```go
func myAppConfig(name, serverUUID string) string {
    return acctest.ConfigProviderBlock() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = %[1]q
}
resource "coolify_dockerfile_application" "test" {
  project_uuid        = coolify_project.test.uuid
  server_uuid         = %[2]q
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
}
`, name, serverUUID)
}
```

Terraform handles creation order and cleanup automatically.

#### Composite import IDs

Resources with composite import IDs (e.g., `project_uuid:name`) need
`ImportStateIdFunc`:

```go
{
    ResourceName:  "coolify_environment.test",
    ImportState:   true,
    ImportStateIdFunc: func(s *terraform.State) (string, error) {
        rs := s.RootModule().Resources["coolify_environment.test"]
        return rs.Primary.Attributes["project_uuid"] + ":" +
               rs.Primary.Attributes["name"], nil
    },
}
```

#### Sensitive fields on import

Fields the API does not return (passwords, keys, secrets) must be
excluded from import verification:

```go
ImportStateVerifyIgnore: []string{"private_key", "postgres_password"},
```

### Available test helpers

| Helper | Purpose |
|--------|---------|
| `acctest.AccTestSkipIfNoTFAcc(t)` | Skip if `TF_ACC` not set |
| `acctest.TestAccPreCheck(t)` | Verify `COOLIFY_ENDPOINT` + `COOLIFY_TOKEN` |
| `acctest.AccTestClient(t)` | Return a configured API client |
| `acctest.AccTestServerUUID(t)` | Discover a usable server UUID |
| `acctest.ConfigProviderBlock()` | Provider HCL from env vars |
| `acctest.TestProtoV6ProviderFactories()` | Provider factory for tests |
| `acctest.RandomWithPrefix(prefix)` | Generate unique resource names |

## Acceptance Test Coverage

### Resources (26 total)

| Resource | Acc Test | Create | Update | Import | Notes |
|----------|----------|--------|--------|--------|-------|
| `coolify_project` | Yes | Yes | Yes | Yes | |
| `coolify_private_key` | Yes | Yes | Yes | Yes | |
| `coolify_s3_storage` | Yes | Yes | Yes | Yes | |
| `coolify_environment` | Yes | Yes | N/A | Yes | All fields RequiresReplace |
| `coolify_dockerfile_application` | Yes | Yes | Yes | Yes | |
| `coolify_postgresql_database` | Yes | Yes | Yes | Yes | |
| `coolify_service` | Yes | Yes | N/A | Yes | All fields RequiresReplace |
| `coolify_deployment` | Yes | Yes | N/A | No | Trigger resource, no update |
| `coolify_environment_variable` | Yes | Yes | Yes | Yes | |
| `coolify_database_backup` | Yes | Yes | Yes | Yes | |
| `coolify_docker_image_application` | Yes | Yes | Yes | Yes | Uses nginx:alpine |
| `coolify_cloud_token` | Yes | Yes | Yes | Yes | |
| `coolify_scheduled_task` | Yes | Yes | Yes | Yes | Attached to dockerfile app |
| `coolify_storage` | Yes | Yes | Yes | Yes | Attached to dockerfile app |
| `coolify_mysql_database` | Yes | Yes | Yes | Yes | Second DB type for coverage |
| `coolify_application` | Yes | Yes | Yes | Yes | Public git with coollabsio/coolify-examples |
| `coolify_private_git_application` | Yes | Yes | Yes | Yes | SSH URL, dummy key |
| `coolify_github_app_application` | N/A | | | | Tested via `coolify_application` variants |
| `coolify_github_app` | Yes | Yes | N/A | Yes | Dummy credentials (metadata only) |
| `coolify_server` | Yes | Yes | Yes | Yes | RFC 5737 IP (192.0.2.1), not reachable |
| `coolify_clickhouse_database` | Yes | Yes | Yes | Yes | |
| `coolify_mariadb_database` | Yes | Yes | Yes | Yes | |
| `coolify_mongodb_database` | Yes | Yes | Yes | Yes | |
| `coolify_redis_database` | Yes | Yes | Yes | Yes | |
| `coolify_dragonfly_database` | Yes | Yes | Yes | Yes | |
| `coolify_keydb_database` | Yes | Yes | Yes | Yes | |

### Data Sources (42 total)

| Data Source | Acc Test | Notes |
|-------------|----------|-------|
| `coolify_version` | Yes | |
| `coolify_health` | Yes | |
| `coolify_team` | Yes | |
| `coolify_teams` | Yes | |
| `coolify_team_members` | Yes | Via team test |
| `coolify_project` | Yes | |
| `coolify_projects` | Yes | |
| `coolify_application` | Yes | Via dockerfile app test |
| `coolify_applications` | Yes | Via dockerfile app test |
| `coolify_database` | Yes | Via postgresql test |
| `coolify_databases` | Yes | Via postgresql test |
| `coolify_environment` | Yes | |
| `coolify_environments` | Yes | |
| `coolify_environment_variables` | Yes | Via envvar test |
| `coolify_private_key` | Yes | Via private_key test |
| `coolify_private_keys` | Yes | Via private_key test |
| `coolify_s3_storage` | Yes | Via s3_storage test |
| `coolify_s3_storages` | Yes | Via s3_storage test |
| `coolify_servers` | Yes | |
| `coolify_server` | Yes | By UUID |
| `coolify_server_resources` | Yes | |
| `coolify_server_domains` | Yes | |
| `coolify_server_validation` | Yes | |
| `coolify_service` | Yes | Via service test |
| `coolify_services` | Yes | Via service test |
| `coolify_deployments` | Yes | Via deployment test |
| `coolify_cloud_token` | Yes | Via cloud_token test |
| `coolify_cloud_tokens` | Yes | Via cloud_token test |
| `coolify_resources` | Yes | |
| `coolify_storages` | Yes | Via storage test |
| `coolify_scheduled_tasks` | Yes | Via scheduled_task test |
| `coolify_task_executions` | Yes | May return empty list |
| `coolify_application_logs` | Yes | May return empty list |
| `coolify_backup_executions` | Yes | May return empty list |
| `coolify_github_app` | Yes | Singular lookup by numeric ID |
| `coolify_github_apps` | Yes | Paired with github_app resource |
| `coolify_github_app_repositories` | Yes | ExpectError with dummy credentials |
| `coolify_github_app_branches` | Yes | ExpectError with dummy credentials |
| `coolify_hetzner_images` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_locations` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_server_types` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_ssh_keys` | Yes | Needs Hetzner token in Coolify |

### Coverage Summary

- **Resources**: 26/26 tested (100%)
- **Data Sources**: 42/42 tested (100%)
- **Total acceptance test functions**: 59

### Testing strategies for edge cases

| Category | Strategy |
|----------|----------|
| **Server registration** | Uses RFC 5737 documentation IP (192.0.2.1); server registers but is not reachable |
| **GitHub App** | Uses dummy credentials (app_id=12345, fake secret/key); CRUD works, API calls to GitHub fail gracefully |
| **GitHub App repos/branches** | Uses `ExpectError` since dummy credentials can't query GitHub |
| **Private Git application** | Uses SSH URL with dummy key; registers metadata without cloning |
| **Hetzner data sources** | Zero-input parameterless calls; pass when Hetzner is configured in Coolify |
| **Runtime data** | application_logs, backup_executions, task_executions may return empty lists; test verifies the list attribute exists |
| **Database variants** | All 8 types (postgresql, mysql, mariadb, clickhouse, mongodb, redis, dragonfly, keydb) tested with identical CRUD pattern |
