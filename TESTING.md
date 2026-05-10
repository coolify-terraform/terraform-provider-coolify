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
| `coolify_application` | - | | | | Needs public Git repo clone (slow) |
| `coolify_docker_compose_application` | - | | | | Needs docker-compose.yml raw content |
| `coolify_private_git_application` | - | | | | Needs SSH key + private repo |
| `coolify_github_app_application` | - | | | | Needs GitHub App configured |
| `coolify_github_app` | - | | | | Needs GitHub App OAuth setup |
| `coolify_server` | - | | | | Needs SSH-accessible target |
| `coolify_clickhouse_database` | - | | | | Same pattern as postgresql |
| `coolify_mariadb_database` | - | | | | Same pattern as postgresql |
| `coolify_mongodb_database` | - | | | | Same pattern as postgresql |
| `coolify_redis_database` | - | | | | Same pattern as postgresql |
| `coolify_dragonfly_database` | - | | | | Same pattern as postgresql |
| `coolify_keydb_database` | - | | | | Same pattern as postgresql |

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
| `coolify_task_executions` | - | Needs a task that has executed |
| `coolify_application_logs` | - | Needs a running app with logs |
| `coolify_backup_executions` | - | Needs a backup that has run |
| `coolify_github_apps` | - | Needs GitHub App configured |
| `coolify_github_app_repositories` | - | Needs GitHub App configured |
| `coolify_github_app_branches` | - | Needs GitHub App configured |
| `coolify_hetzner_images` | - | Needs Hetzner token |
| `coolify_hetzner_locations` | - | Needs Hetzner token |
| `coolify_hetzner_server_types` | - | Needs Hetzner token |
| `coolify_hetzner_ssh_keys` | - | Needs Hetzner token |

### Coverage Summary

- **Resources**: 15/27 tested (56%), 12 untested require external
  infrastructure or are database variants of the tested postgresql/mysql pattern
- **Data Sources**: 31/41 tested (76%), 10 untested require external
  infrastructure (GitHub App, Hetzner, running app logs, executed backups/tasks)

### Why some resources/data sources are not tested

| Category | Resources/Data Sources | Reason |
|----------|----------------------|--------|
| **GitHub App** | github_app, github_app_application, github_apps, github_app_repositories, github_app_branches | Requires a registered GitHub App with OAuth credentials |
| **Hetzner** | hetzner_images, hetzner_locations, hetzner_server_types, hetzner_ssh_keys | Requires a Hetzner cloud token configured in Coolify |
| **Server registration** | server | Requires an SSH-accessible target machine |
| **Git-based apps** | application (public git), private_git_application, docker_compose_application | Requires actual Git repo clone (slow, flaky in CI) |
| **Runtime data** | application_logs, backup_executions, task_executions | Requires a running application/executed backup/task |
| **Other DB types** | clickhouse, mariadb, mongodb, redis, dragonfly, keydb | Identical pattern to postgresql; tested via unit tests |
