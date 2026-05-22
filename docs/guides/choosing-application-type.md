---
page_title: "Choosing an Application Type"
subcategory: "Getting Started"
description: |-
  How to pick the right application resource for your deployment method.
---

# Choosing an Application Type

The provider offers five application resources, each tailored to a
different deployment method. This guide helps you pick the right one.

## Decision Flowchart

```
Start here: What do you have?
│
├─ Source code in a Git repository
│  │
│  ├─ Public repository (no authentication needed)
│  │  │
│  │  ├─ Has a Dockerfile ─────► coolify_application (build_pack = "dockerfile")
│  │  └─ No Dockerfile ────────► coolify_application (build_pack = "nixpacks")
│  │
│  └─ Private repository
│     │
│     ├─ SSH deploy key ────────► coolify_application_private_git
│     └─ GitHub App installed ──► coolify_application_github_app
│
├─ A Dockerfile (no Git repo) ──► coolify_application_dockerfile
│
└─ A pre-built Docker image ────► coolify_application_docker_image
```

## Comparison Table

| Resource | Source | Auth | Build System | Best For |
|----------|--------|------|-------------|----------|
| [`coolify_application`](/docs/resources/application) | Public Git repo | None | nixpacks, dockerfile, buildpacks | Open-source apps, public repos |
| [`coolify_application_private_git`](/docs/resources/application_private_git) | Private Git repo | SSH key (`coolify_private_key`) | nixpacks, dockerfile, buildpacks | Private repos without GitHub Apps |
| [`coolify_application_github_app`](/docs/resources/application_github_app) | GitHub repo | GitHub App (`coolify_github_app`) | nixpacks, dockerfile, buildpacks | GitHub-hosted repos with fine-grained permissions |
| [`coolify_application_dockerfile`](/docs/resources/application_dockerfile) | Inline Dockerfile | None | Dockerfile | Custom images without a Git repo, Terraform-managed Dockerfiles |
| [`coolify_application_docker_image`](/docs/resources/application_docker_image) | Docker registry | None (public) or registry auth | None (pre-built) | Pre-built images from Docker Hub, GHCR, ECR |

## Quick Examples

### Public Git repository

```hcl
resource "coolify_application" "web" {
  name           = "my-app"
  project_uuid   = coolify_project.main.uuid
  server_uuid    = data.coolify_server.prod.uuid
  git_repository = "https://github.com/myorg/myapp"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}
```

### Private Git repository (SSH key)

```hcl
resource "coolify_application_private_git" "api" {
  name             = "api-server"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = data.coolify_server.prod.uuid
  git_repository   = "git@github.com:myorg/api-server.git"
  git_branch       = "main"
  private_key_uuid = coolify_private_key.deploy.uuid
  build_pack       = "dockerfile"
  ports_exposes    = "8080"
}
```

### GitHub App integration

```hcl
resource "coolify_application_github_app" "app" {
  name            = "frontend"
  project_uuid    = coolify_project.main.uuid
  server_uuid     = data.coolify_server.prod.uuid
  github_app_uuid = coolify_github_app.org.uuid
  git_repository  = "https://github.com/myorg/frontend"
  git_branch      = "main"
  build_pack      = "nixpacks"
  ports_exposes   = "3000"
}
```

### Inline Dockerfile (no Git repo)

```hcl
resource "coolify_application_dockerfile" "custom" {
  name         = "custom-nginx"
  project_uuid = coolify_project.main.uuid
  server_uuid  = data.coolify_server.prod.uuid
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM nginx:alpine
    COPY index.html /usr/share/nginx/html/
    EXPOSE 80
  DOCKERFILE
  )
  ports_exposes = "80"
}
```

~> **Note:** `dockerfile_location` expects base64-encoded Dockerfile
content, not a file path. Use `base64encode()`.

### Pre-built Docker image

```hcl
resource "coolify_application_docker_image" "redis" {
  name          = "redis-cache"
  project_uuid  = coolify_project.main.uuid
  server_uuid   = data.coolify_server.prod.uuid
  docker_image  = "redis:7-alpine"
  ports_exposes = "6379"
}
```

## Can I Switch Later?

Changing from one application type to another requires destroying the
existing resource and creating a new one. The resource types have
different schemas and map to different Coolify configurations internally.

In Terraform terms: you must remove the old resource block, add the new
one, and re-import or let Terraform create a fresh resource. The
application UUID will change.

If you are unsure which type to start with, `coolify_application` with
`build_pack = "nixpacks"` is the most flexible starting point. Nixpacks
auto-detects most frameworks (Node.js, Python, Go, Rust, Ruby, PHP,
Java, .NET) and falls back to Dockerfile if one exists in the repo.

## Build Pack Options

All Git-based application types support these build packs:

| Build Pack | How It Works |
|-----------|-------------|
| `nixpacks` | Auto-detects your framework and generates a build plan. Supports 20+ languages. Best default choice. |
| `dockerfile` | Uses a `Dockerfile` in your repository. Full control over the build. |
| `static` | Serves static files directly (HTML, CSS, JS). No build step. |

Set the build pack with the `build_pack` attribute:

```hcl
resource "coolify_application" "web" {
  # ...
  build_pack = "nixpacks"  # or "dockerfile" or "static"
}
```
