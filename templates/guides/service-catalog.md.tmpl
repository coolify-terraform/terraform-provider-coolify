---
page_title: "Service Catalog"
subcategory: "Guides"
description: |-
  Deploy one-click services from the Coolify catalog with Terraform.
---

# Service Catalog

Coolify includes a catalog of pre-configured services that deploy as
Docker Compose stacks with a single resource. This guide covers how to
use them with the `coolify_service` resource.

## How Services Work

Each service in the catalog is a Docker Compose template maintained by
the Coolify project. When you create a `coolify_service` with a given
`type`, Coolify:

1. Loads the corresponding Compose template
2. Creates all containers defined in the template
3. Configures networking and proxy rules
4. Starts the service stack

The entire stack is managed as one Terraform resource. You cannot
individually control containers within a service.

## Creating a Service

```hcl
resource "coolify_service" "monitoring" {
  name             = "uptime-kuma"
  type             = "uptime-kuma"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = data.coolify_server.prod.uuid
  environment_name = "production"
}
```

The `type` field must exactly match one of the catalog entries. The
`name` field is your label for the service.

## Common Services

Below are popular services from the catalog with working examples.

### Monitoring and Uptime

```hcl
# Uptime monitoring with status pages
resource "coolify_service" "uptime" {
  name             = "uptime-kuma"
  type             = "uptime-kuma"
  project_uuid     = coolify_project.ops.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}

# Metrics visualization
resource "coolify_service" "grafana" {
  name             = "grafana"
  type             = "grafana"
  project_uuid     = coolify_project.ops.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### Analytics

```hcl
# Privacy-friendly web analytics
resource "coolify_service" "plausible" {
  name             = "plausible"
  type             = "plausible"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}

# Lightweight analytics alternative
resource "coolify_service" "umami" {
  name             = "umami"
  type             = "umami"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### Object Storage

```hcl
# S3-compatible object storage
resource "coolify_service" "minio" {
  name             = "minio"
  type             = "minio"
  project_uuid     = coolify_project.infra.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### Git Hosting

```hcl
# Lightweight Git hosting
resource "coolify_service" "gitea" {
  name             = "gitea"
  type             = "gitea"
  project_uuid     = coolify_project.dev.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### CMS and Content

```hcl
# Ghost blogging platform
resource "coolify_service" "ghost" {
  name             = "ghost"
  type             = "ghost"
  project_uuid     = coolify_project.content.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}

# Headless CMS
resource "coolify_service" "directus" {
  name             = "directus"
  type             = "directus"
  project_uuid     = coolify_project.content.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### Automation and Low-Code

```hcl
# Workflow automation
resource "coolify_service" "n8n" {
  name             = "n8n"
  type             = "n8n"
  project_uuid     = coolify_project.ops.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}

# No-code database
resource "coolify_service" "nocodb" {
  name             = "nocodb"
  type             = "nocodb"
  project_uuid     = coolify_project.tools.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

### Development Tools

```hcl
# Browser-based VS Code
resource "coolify_service" "code_server" {
  name             = "code-server"
  type             = "code-server"
  project_uuid     = coolify_project.dev.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}
```

## Finding Available Types

The catalog evolves with each Coolify release. To see what your instance
supports:

**Option A: Coolify UI.** Navigate to **New Resource > Service** to see
the full catalog with descriptions.

**Option B: Coolify source.** The catalog is defined in the
[Coolify repository](https://github.com/coollabsio/coolify) under
`app/Models/Service.php` and the `templates/service-templates.json`
directory.

## Customizing Services

Services accept environment variables through `coolify_environment_variable`
with the `service_uuid` attribute:

```hcl
resource "coolify_service" "plausible" {
  name             = "plausible"
  type             = "plausible"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
}

resource "coolify_environment_variable" "plausible_url" {
  service_uuid = coolify_service.plausible.uuid
  key          = "BASE_URL"
  value        = "https://analytics.example.com"
}
```

The available environment variables depend on the specific service. Check
the service's official documentation or the Compose template in the Coolify
source for supported variables.

## Limitations

- **Managed as a unit.** All containers in a service start, stop, and
  delete together. You cannot manage individual containers.
- **Changes force replacement.** Modifying `name`, `description`, or
  `environment_name` destroys and recreates the entire service. Update
  these fields only when you can tolerate downtime.
- **Type is immutable.** You cannot change a service's `type` after
  creation. Destroy and recreate with the new type.
- **Connect to Docker network.** Set `connect_to_docker_network = true`
  if the service containers need to reach other resources in the project
  network (databases, applications).

```hcl
resource "coolify_service" "grafana" {
  name                       = "grafana"
  type                       = "grafana"
  project_uuid               = coolify_project.ops.uuid
  server_uuid                = var.server_uuid
  environment_name           = "production"
  connect_to_docker_network  = true  # reach databases in the same project
}
```
