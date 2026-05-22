---
page_title: "Connecting Resources"
subcategory: "Guides"
description: |-
  How Coolify networking works and how to connect applications to databases and other services.
---

# Connecting Resources

Coolify places all resources within the same project on a shared Docker
network. This guide explains how that networking works and how to wire
applications to databases and other services.

## How Coolify Networking Works

Every Coolify project gets its own Docker network. All resources (apps,
databases, services) within that project are attached to the same network.
This means:

- **Containers can reach each other by resource name.** If your database
  is named `my-postgres`, other containers in the same project can connect
  to it at hostname `my-postgres`.
- **No port exposure needed.** Databases stay private (`is_public = false`)
  and applications connect over the internal network.
- **Cross-project networking is not supported.** Resources in different
  projects cannot reach each other by hostname. If two resources need to
  communicate, place them in the same project.

```
┌─────────────────── Project: "my-app" ──────────────────┐
│                                                         │
│  ┌───────────────┐       ┌────────────────────────┐     │
│  │  PostgreSQL    │◄──────│  Node.js Application   │     │
│  │  name: "mydb"  │       │  name: "web"           │     │
│  │  port 5432     │       │  DATABASE_URL=          │     │
│  │  is_public=    │       │    ...@mydb:5432/app   │     │
│  │   false        │       │                        │     │
│  └───────────────┘       └────────────────────────┘     │
│         ▲                                               │
│         │ Docker network: hostname = resource name      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Full Working Example

This example deploys a Node.js app connected to a PostgreSQL database:

```hcl
resource "coolify_project" "myapp" {
  name = "my-application"
}

resource "coolify_database_postgresql" "db" {
  name              = "app-db"
  project_uuid      = coolify_project.myapp.uuid
  server_uuid       = var.server_uuid
  image             = "postgres:16"
  postgres_user     = "app"
  postgres_password = var.db_password
  postgres_db       = "myapp"
  is_public         = false  # only reachable within the project network
}

resource "coolify_application" "web" {
  name           = "web"
  project_uuid   = coolify_project.myapp.uuid
  server_uuid    = var.server_uuid
  git_repository = "https://github.com/myorg/myapp"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

# Wire the app to the database using an environment variable.
# "app-db" is the database resource name, which resolves as a hostname
# on the shared Docker network.
resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.web.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_database_postgresql.db.postgres_user}:${coolify_database_postgresql.db.postgres_password}@${coolify_database_postgresql.db.name}:5432/${coolify_database_postgresql.db.postgres_db}"
  is_build         = false
  is_preview       = false
}
```

**How the hostname works:** the `name` attribute of `coolify_database_postgresql.db`
is `"app-db"`. Inside the Docker network, other containers resolve `app-db` to
the database container's IP address. The connection string uses this name
instead of an IP.

## Connection String Patterns

Use these patterns to connect to each database type. Replace `<name>` with
your database resource's `name` attribute.

### PostgreSQL

```
postgresql://<user>:<password>@<name>:5432/<database>
```

```hcl
value = "postgresql://${coolify_database_postgresql.db.postgres_user}:${coolify_database_postgresql.db.postgres_password}@${coolify_database_postgresql.db.name}:5432/${coolify_database_postgresql.db.postgres_db}"
```

### MySQL / MariaDB

```
mysql://<user>:<password>@<name>:3306/<database>
```

```hcl
# MySQL
value = "mysql://${coolify_database_mysql.db.mysql_user}:${coolify_database_mysql.db.mysql_password}@${coolify_database_mysql.db.name}:3306/${coolify_database_mysql.db.mysql_database}"

# MariaDB
value = "mysql://${coolify_database_mariadb.db.mariadb_user}:${coolify_database_mariadb.db.mariadb_password}@${coolify_database_mariadb.db.name}:3306/${coolify_database_mariadb.db.mariadb_database}"
```

### Redis / KeyDB / Dragonfly

```
redis://:<password>@<name>:6379
```

```hcl
# Redis
value = "redis://:${coolify_database_redis.cache.redis_password}@${coolify_database_redis.cache.name}:6379"

# KeyDB
value = "redis://:${coolify_database_keydb.cache.keydb_password}@${coolify_database_keydb.cache.name}:6379"

# Dragonfly
value = "redis://:${coolify_database_dragonfly.cache.dragonfly_password}@${coolify_database_dragonfly.cache.name}:6379"
```

### MongoDB

```
mongodb://<user>:<password>@<name>:27017/<database>
```

```hcl
value = "mongodb://${coolify_database_mongodb.db.mongo_initdb_root_username}:${coolify_database_mongodb.db.mongo_initdb_root_password}@${coolify_database_mongodb.db.name}:27017/${coolify_database_mongodb.db.mongo_initdb_database}"
```

### ClickHouse

```
clickhouse://<user>:<password>@<name>:8123/<database>
```

```hcl
value = "clickhouse://${coolify_database_clickhouse.analytics.clickhouse_admin_user}:${coolify_database_clickhouse.analytics.clickhouse_admin_password}@${coolify_database_clickhouse.analytics.name}:8123/${coolify_database_clickhouse.analytics.clickhouse_db_name}"
```

## Public vs Private Access

| Setting | `is_public = false` (default) | `is_public = true` |
|---------|------------------------------|-------------------|
| Internal access | Yes, via resource name | Yes, via resource name |
| External access | No | Yes, via server IP + `public_port` |
| Security | Recommended for production | Use when external tools need direct access |

Set `is_public = true` only when something outside the Coolify project
needs direct database access (monitoring tools, database GUIs, migration
scripts running on your laptop):

```hcl
resource "coolify_database_postgresql" "db" {
  # ...
  is_public   = true
  public_port = 15432  # exposed on the server's public IP
}
```

## Apps Talking to Apps

Applications within the same project can also reach each other by name.
For example, a frontend app calling a backend API:

```hcl
resource "coolify_environment_variable" "api_url" {
  application_uuid = coolify_application.frontend.uuid
  key              = "API_URL"
  value            = "http://${coolify_application.backend.name}:8080"
  is_build         = false
}
```

## Common Mistakes

### Using an IP address instead of the resource name

The resource name resolves inside the Docker network. The server's IP
address does not reach the container unless `is_public = true`.

```hcl
# Wrong: 192.168.1.100 does not resolve to the container
value = "postgresql://app:pass@192.168.1.100:5432/myapp"

# Correct: use the resource name
value = "postgresql://app:pass@${coolify_database_postgresql.db.name}:5432/myapp"
```

### Database in a different project

Resources in different projects are on different Docker networks. They
cannot reach each other by hostname. Move both resources into the same
project, or set `is_public = true` on the database and connect via the
server's public IP (less secure).

### Forgetting the environment variable

Creating a database and an application in the same project does not
automatically connect them. You must pass the connection string via
`coolify_environment_variable`. The application code reads it from
the environment at runtime.
