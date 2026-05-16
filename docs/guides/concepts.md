---
page_title: "Core Concepts"
subcategory: "Getting Started"
description: |-
  Understand the Coolify resource model and how the provider maps to it.
---

# Core Concepts

## Resource hierarchy

Coolify organizes infrastructure in a hierarchy:

```
Server (physical or VM with Docker)
  └── Project (logical grouping)
       └── Environment (production, staging, dev)
            ├── Application (web app, API, worker)
            ├── Database (PostgreSQL, MySQL, Redis, etc.)
            └── Service (one-click apps from the catalog)
```

The Terraform provider maps each level to a resource or data source.

## Resource types

### Infrastructure

| Resource | Purpose |
|----------|---------|
| `coolify_project` | Logical group for applications and databases |
| `coolify_environment` | Environment within a project (production, staging) |
| `coolify_server` | A Docker host registered with Coolify |
| `coolify_private_key` | SSH key for server access or Git clone |
| `coolify_cloud_token` | Hetzner/cloud provider API token |
| `coolify_github_app` | GitHub App integration for repository access |

### Applications

Five application types cover different deployment methods:

| Resource | Source |
|----------|--------|
| `coolify_application` | Public Git repository |
| `coolify_private_git_application` | Private Git repository (SSH key) |
| `coolify_github_app_application` | GitHub App integration |
| `coolify_dockerfile_application` | Dockerfile (no Git repo required) |
| `coolify_docker_image_application` | Docker image (Docker Hub, GHCR) |

All application resources support configurable `timeouts` for long builds.

### Databases

Eight database engines are supported:

`coolify_postgresql_database`, `coolify_mysql_database`,
`coolify_mariadb_database`, `coolify_redis_database`,
`coolify_mongodb_database`, `coolify_clickhouse_database`,
`coolify_keydb_database`, `coolify_dragonfly_database`

All database resources share the same base schema (`project_uuid`,
`server_uuid`, `image`, `is_public`, `public_port`).

### Operations

| Resource | Purpose |
|----------|---------|
| `coolify_environment_variable` | Inject config into apps, services, or databases |
| `coolify_storage` | Persistent volume mount for apps, services, or databases |
| `coolify_scheduled_task` | Cron-based task on apps or services |
| `coolify_deployment` | Trigger a deploy (with force-redeploy via `triggers`) |
| `coolify_database_backup` | Schedule automated backups with cron syntax |
| `coolify_service` | Deploy one-click services from the Coolify catalog |

## Data sources

Every resource has a corresponding data source for reading existing state.
Plural data sources (e.g., `coolify_projects`) return all resources of that
type. Additional read-only data sources include:

- `coolify_server_resources`: list all resources deployed on a server
- `coolify_server_domains`: list all domains configured on a server
- `coolify_environment_variables`: list env vars for an app, service, or database
- `coolify_version`: read the Coolify instance version
- `coolify_team`: read team info and members

## Import

All resources support `terraform import`. Most use a single UUID:

```bash
terraform import coolify_project.main <uuid>
```

Resources with composite IDs use colon-separated values:

```bash
terraform import coolify_database_backup.daily <db-uuid>:<backup-id>
terraform import coolify_environment_variable.key application:<app-uuid>:<env-uuid>
```

See the [Import Guide](import) for full details.

## Immutable fields

Some fields force resource replacement when changed. These are marked with
`RequiresReplace` in the schema:

- `project_uuid`, `server_uuid`, `environment_name` on all
  application and database resources

Plan output will show `# forces replacement` for these fields.

## Next steps

Follow the [Quick Start](quickstart) to deploy your first application in
under five minutes.
