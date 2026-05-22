---
page_title: "Build Lifecycle"
subcategory: "Guides"
description: |-
  Understand what happens after terraform apply: build queuing, status tracking, and deployment waiting.
---

# Build Lifecycle

When `terraform apply` creates or updates an application, Coolify
**queues a build**. Terraform's job ends when the Coolify API confirms
the resource exists. The build, container start, and health check happen
asynchronously.

This guide explains the lifecycle, how to monitor it, and how to wait
for completion in CI/CD pipelines.

## What Happens After Apply

```
terraform apply
  │
  ├─ 1. Provider calls POST /api/v1/applications
  │     Coolify creates the resource and returns a UUID
  │
  ├─ 2. Provider calls GET /api/v1/applications/{uuid}
  │     Reads back the full resource state
  │
  └─ 3. Terraform writes state ✓  (apply is "done" here)

Meanwhile, in Coolify:
  │
  ├─ 4. Build is queued on the server
  ├─ 5. Source is cloned, image is built (nixpacks/Dockerfile)
  ├─ 6. Container starts on the server
  └─ 7. Health check passes → status becomes "running"
```

**Key point:** steps 4-7 happen after Terraform finishes. A successful
`terraform apply` means the resource exists in Coolify, not that the
application is live and serving traffic.

## Checking Application Status

### The `status` attribute

All application resources have a computed `status` attribute:

```hcl
output "app_status" {
  value = coolify_application.web.status
}
```

Common status values:

| Status | Meaning |
|--------|---------|
| `running` | Container is running and healthy |
| `building` | Build is in progress |
| `stopped` | Container is stopped |
| `exited` | Container exited (crash or completed) |
| `restarting` | Container is restarting |

### Reading logs

Use the `coolify_application_logs` data source to check build output:

```hcl
data "coolify_application_logs" "build" {
  application_uuid = coolify_application.web.uuid
}

output "recent_logs" {
  value = data.coolify_application_logs.build.logs
}
```

### Checking deployment history

Use the `coolify_deployments` data source to see all deployments:

```hcl
data "coolify_deployments" "web" {
  application_uuid = coolify_application.web.uuid
}
```

## Waiting for Deployment in CI

For CI/CD pipelines where you need to confirm the deployment succeeded
before proceeding, use the `coolify_deployment` resource with
`wait_for_completion`:

```hcl
resource "coolify_deployment" "web" {
  application_uuid    = coolify_application.web.uuid
  wait_for_completion = true

  triggers = {
    # Change this value to force a new deployment
    deploy_version = "v1.2.3"
  }

  timeouts {
    create = "15m"  # wait up to 15 minutes for the build
  }
}
```

When `wait_for_completion = true`, the provider polls the deployment
status until it reaches `finished` or `error`. If the build fails or
the timeout expires, Terraform reports an error.

## Force Redeployment

### Using `coolify_deployment` with triggers

The `triggers` map forces a new deployment whenever any value changes:

```hcl
resource "coolify_deployment" "web" {
  application_uuid = coolify_application.web.uuid

  triggers = {
    # Changing any value here triggers a new deployment
    version    = "v1.2.3"
    config_sha = sha256(file("config.yml"))
  }
}
```

### Using `redeploy_on_update`

Application resources have a `redeploy_on_update` attribute that
triggers a restart whenever any configuration field changes:

```hcl
resource "coolify_application" "web" {
  # ...
  redeploy_on_update = true
}
```

**When to use which:**

| Approach | Use case |
|----------|----------|
| `coolify_deployment` with `triggers` | Explicit deploy control, CI/CD pipelines, deploy-on-demand |
| `redeploy_on_update = true` | Auto-restart when any Terraform config changes |

These are independent. `redeploy_on_update` reacts to changes in the
application resource itself. `coolify_deployment` is a separate resource
you control with trigger values.

## Common Build Failures

### Nixpacks cannot detect framework

Nixpacks auto-detects frameworks by looking for configuration files
(`package.json`, `requirements.txt`, `go.mod`, etc.). If your project
structure is non-standard:

- Set `base_directory` to the directory containing the framework's config file
- Switch to `build_pack = "dockerfile"` and provide your own Dockerfile

### Port mismatch

Coolify routes traffic to the port specified in `ports_exposes`. If your
application listens on a different port, it will appear unhealthy:

```hcl
resource "coolify_application" "web" {
  # ...
  ports_exposes = "8080"  # must match what your app actually listens on
}
```

### "Application created but refresh failed"

This error means Coolify returned a UUID for the new application, but
the provider could not read it back via GET. Common causes:

1. **Server not SSH-reachable**: Coolify SSHs into servers to manage
   Docker. If the server is unreachable, the application is created in
   the database but cannot be read back.
2. **Server still provisioning**: if you just created the server with
   `coolify_server_hetzner`, wait for validation to complete before
   deploying applications.

Fix: ensure the server shows `is_usable = true`, then run
`terraform apply` again. The partial state is saved, so Terraform
picks up where it left off.

## Start, Stop, and Restart

Use `coolify_resource_action` to control application lifecycle:

```hcl
# Stop an application
resource "coolify_resource_action" "stop_app" {
  resource_uuid = coolify_application.web.uuid
  resource_type = "application"
  action        = "stop"
}

# Restart an application
resource "coolify_resource_action" "restart_app" {
  resource_uuid = coolify_application.web.uuid
  resource_type = "application"
  action        = "restart"
}
```
