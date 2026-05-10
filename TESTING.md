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

```bash
# Start a local Coolify instance
docker compose up -d

# Complete initial setup at http://localhost:8000
# Create an API token from Settings > API

# Set environment variables
export COOLIFY_ENDPOINT="http://localhost:8000"
export COOLIFY_TOKEN="<your-api-token>"

# Optional: set a specific server UUID (otherwise auto-discovered)
export COOLIFY_SERVER_UUID="<server-uuid>"
```

### Running acceptance tests

```bash
# Run all acceptance tests
make testacc

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

### Resources (27 total)

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
| `coolify_docker_compose_application` | Yes | Yes | Yes | Yes | Inline compose YAML |
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

### Data Sources (41 total)

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
| `coolify_github_apps` | Yes | Paired with github_app resource |
| `coolify_github_app_repositories` | Yes | ExpectError with dummy credentials |
| `coolify_github_app_branches` | Yes | ExpectError with dummy credentials |
| `coolify_hetzner_images` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_locations` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_server_types` | Yes | Needs Hetzner token in Coolify |
| `coolify_hetzner_ssh_keys` | Yes | Needs Hetzner token in Coolify |

### Coverage Summary

- **Resources**: 27/27 tested (100%)
- **Data Sources**: 41/41 tested (100%)
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
