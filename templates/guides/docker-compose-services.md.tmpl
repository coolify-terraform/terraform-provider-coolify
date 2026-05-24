---
page_title: "Deploy Docker Compose Stacks"
subcategory: "Guides"
description: "How to deploy custom Docker Compose stacks via Terraform using coolify_service with docker_compose_raw."
---

# Deploy Docker Compose Stacks

Coolify can deploy any Docker Compose stack, not just services from its built-in catalog. This guide shows how to deploy a custom `docker-compose.yml` with Terraform.

## Quick Start

```hcl
resource "coolify_project" "myproject" {
  name = "my-stack"
}

resource "coolify_service" "mystack" {
  name         = "redis-stack"
  project_uuid = coolify_project.myproject.uuid
  server_uuid  = "your-server-uuid"

  docker_compose_raw = file("docker-compose.yml")
}
```

The provider accepts **plain YAML**. Base64 encoding is handled automatically before sending to the Coolify API. If you prefer, you can also pass pre-encoded content with `base64encode()`, and the provider will detect it and avoid double-encoding.

## Catalog vs Custom Compose

The `coolify_service` resource supports two creation modes:

| Mode | Attribute | Use case |
|------|-----------|----------|
| **Catalog** | `type = "plausible"` | Deploy a pre-built stack from Coolify's service catalog |
| **Custom Compose** | `docker_compose_raw = file(...)` | Deploy any Docker Compose stack |

These are **mutually exclusive**. You cannot set both `type` and `docker_compose_raw` in the same resource.

## Dynamic Compose with templatefile

Use Terraform's `templatefile()` to inject variables into your compose configuration:

```hcl
resource "coolify_service" "app" {
  name         = "my-app"
  project_uuid = coolify_project.myproject.uuid
  server_uuid  = var.server_uuid

  docker_compose_raw = templatefile("docker-compose.yml.tpl", {
    app_image  = var.app_image
    app_port   = var.app_port
    redis_pass = var.redis_password
  })
}
```

Where `docker-compose.yml.tpl` is:

```yaml
version: "3"
services:
  app:
    image: ${app_image}
    ports:
      - "${app_port}:8080"
    environment:
      REDIS_URL: "redis://:${redis_pass}@redis:6379"
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${redis_pass}
```

## Managing Environment Variables

For sensitive values, use `coolify_environment_variable` instead of hardcoding them in the compose YAML:

```hcl
resource "coolify_environment_variable" "db_password" {
  resource_uuid = coolify_service.mystack.uuid
  resource_type = "service"
  key           = "DB_PASSWORD"
  value         = var.db_password
  is_build      = false
}
```

Then reference `${DB_PASSWORD}` in your compose file. Coolify injects these at deploy time.

## Customizing a Catalog Service

You can create a service from the catalog and then customize its compose on a subsequent apply:

```hcl
# First apply: creates from catalog
resource "coolify_service" "grafana" {
  type         = "grafana"
  project_uuid = coolify_project.myproject.uuid
  server_uuid  = var.server_uuid
}

# After the first apply, you can import the generated compose,
# modify it, and set docker_compose_raw on subsequent applies
# via the update path (PATCH).
```

## Token Permissions

The `docker_compose_raw` and `docker_compose` fields require an API token with `read:sensitive` or `root` permissions. Without this, these fields appear empty after import or read-back, which may cause unexpected diffs.

Check your token permissions in the Coolify dashboard under **Security > API Tokens**.

## Common Pitfalls

- **Compose YAML must be valid.** Coolify validates the compose for security (command injection checks). Invalid YAML or dangerous volume mounts will be rejected.
- **Named volumes work.** Coolify creates Docker named volumes on the target server.
- **Host-path bind mounts depend on the server.** Paths like `/data/myapp` must exist on the target server.
- **The API returns decoded YAML.** The provider preserves your original input in state to avoid diffs between your raw YAML and the API's re-formatted version.
