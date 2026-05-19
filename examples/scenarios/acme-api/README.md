# ACME Corp Order Processing API

An intermediate Coolify scenario that deploys a complete order processing
backend as microservices.

## The Story

ACME Corp's e-commerce platform needs an order processing backend. The REST
API handles incoming orders, a background worker processes payments and sends
notifications. PostgreSQL stores orders, Redis handles job queues and caching.

All services are deployed to a single Coolify server, connected via internal
Docker networking, and managed entirely through Terraform.

## What Gets Created

| # | Resource | Type | Purpose |
|---|----------|------|---------|
| 1 | `coolify_project.acme` | Project | Groups all order-processing resources |
| 2 | `coolify_database_postgresql.orders` | Database | Stores orders, customers, payments |
| 3 | `coolify_database_redis.queue` | Database | Job queues and response caching |
| 4 | `coolify_dockerfile_application.api` | Application | REST API built from a Dockerfile |
| 5 | `coolify_docker_image_application.worker` | Application | Background worker from a Docker image |
| 6 | `coolify_environment_variable.api_db_url` | Env var | Connects the API to PostgreSQL |
| 7 | `coolify_environment_variable.api_redis_url` | Env var | Connects the API to Redis |
| 8 | `coolify_environment_variable.worker_db_url` | Env var | Connects the worker to PostgreSQL |
| 9 | `coolify_environment_variable.worker_redis_url` | Env var | Connects the worker to Redis |
| 10 | `coolify_scheduled_task.cleanup` | Scheduled task | Nightly cleanup of old orders |
| 11 | `coolify_database_backup.orders` | Backup | Daily local backup of the orders DB |

## Architecture

```
                         ┌─────────────────────────────────────────────┐
                         │              Coolify Server                 │
                         │                                             │
    HTTP requests ──────►│  ┌──────────────────┐                       │
                         │  │   API (Dockerfile)│──┐                   │
                         │  │   :3000           │  │                   │
                         │  └──────────────────┘  │                   │
                         │           │             │                   │
                         │      env vars          env vars             │
                         │           │             │                   │
                         │           ▼             ▼                   │
                         │  ┌──────────┐   ┌──────────────┐           │
                         │  │PostgreSQL│   │    Redis      │           │
                         │  │  :5432   │   │    :6379      │           │
                         │  └──────────┘   └──────────────┘           │
                         │           ▲             ▲                   │
                         │      env vars          env vars             │
                         │           │             │                   │
                         │  ┌──────────────────┐  │                   │
                         │  │ Worker (Docker   │──┘                   │
                         │  │  Image) :8080    │                       │
                         │  └──────────────────┘                       │
                         │                                             │
                         │  ┌──────────────────┐  ┌────────────────┐  │
                         │  │ Scheduled Task:  │  │ Daily Backup:  │  │
                         │  │ nightly-cleanup  │  │ orders DB      │  │
                         │  └──────────────────┘  └────────────────┘  │
                         └─────────────────────────────────────────────┘
```

## Prerequisites

1. A running [Coolify](https://coolify.io/) instance (v4+)
2. An API token (Security > API Tokens)
3. A registered server UUID (Settings → Servers)
4. [Terraform](https://developer.hashicorp.com/terraform/install) ≥ 1.0

## Usage

```bash
# Initialize providers
terraform init

# Preview the plan
terraform plan \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"

# Apply
terraform apply \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```

Or use a `terraform.tfvars` file:

```hcl
coolify_endpoint = "https://coolify.example.com"
coolify_token    = "your-api-token"
server_uuid      = "your-server-uuid"
```

## How It Works

1. **Project**: a `coolify_project` groups all resources under "acme-orders".
2. **Databases**: PostgreSQL stores order data; Redis provides job queues and
   caching. Both are internal-only (not publicly exposed).
3. **API application**: built from a Dockerfile (`coolify_dockerfile_application`).
   Environment variables inject the database and Redis connection strings.
4. **Worker application**: deployed from a pre-built Docker image
   (`coolify_docker_image_application`). Shares the same database and Redis
   connections so it can pick up jobs enqueued by the API.
5. **Scheduled task**: a `coolify_scheduled_task` runs a nightly cleanup
   command inside the API container.
6. **Backup**: a `coolify_database_backup` schedules daily local backups of
   the PostgreSQL database.

## Clean Up

```bash
terraform destroy \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```

## Key Concepts

### Dockerfile vs Docker Image Deployment

- **`coolify_dockerfile_application`**: Coolify builds the image from a
  Dockerfile you provide. Use this when you have source code and a build step.
- **`coolify_docker_image_application`**: Coolify pulls a pre-built image
  from a registry (Docker Hub, GHCR, etc.). Use this for off-the-shelf images
  or images built by your CI/CD pipeline.

### Scheduled Tasks

A `coolify_scheduled_task` runs a shell command inside an application container
on a cron schedule. Tasks are useful for maintenance jobs (cleanup, reports,
health checks) without deploying a separate cron container.

### Cross-Service Communication

In Coolify, a resource's `name` doubles as its Docker container name and
internal DNS hostname. Services on the same server can reach each other
using `<name>:<port>`, so no public exposure is needed. This scenario connects
the API and worker to PostgreSQL and Redis entirely through internal
environment variables.
