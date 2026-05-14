# ACME Corp Marketing Website

## The Story

ACME Corp is migrating their marketing website off Heroku. The site is a
Node.js application backed by a PostgreSQL database that stores page content,
blog posts, and lead-capture form submissions. The team wants infrastructure
as code so every change is reviewed in a pull request, and they need automated
daily backups shipped to S3-compatible storage so the marketing data is never
at risk.

This scenario provisions the entire stack (project, database, application,
environment wiring, and an optional backup schedule that targets an existing
UI-managed S3 storage) in a single `terraform apply`.

## What Gets Created

| # | Resource | Purpose |
|---|----------|---------|
| 1 | `coolify_project.acme` | Logical grouping for all ACME resources |
| 2 | `coolify_postgresql_database.content` | PostgreSQL 16 database for page content |
| 3 | `coolify_application.website` | Node.js marketing site from Git |
| 4 | `coolify_environment_variable.database_url` | Connects the app to the database |
| 5 | `coolify_environment_variable.node_env` | Sets NODE_ENV=production for builds |
| 6 | `coolify_database_backup.daily` | Daily backup schedule using an existing UI-managed S3 storage *(optional)* |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Coolify Project                    │
│                  "acme-website"                     │
│                                                     │
│  ┌──────────────────┐    ┌───────────────────────┐  │
│  │   PostgreSQL 16   │    │   Node.js Application │  │
│  │   "acme-content"  │◄───│   "acme-website"      │  │
│  │                   │    │                       │  │
│  │   postgres:16     │    │   nixpacks build      │  │
│  │   port 5432       │    │   port 3000           │  │
│  └────────┬──────────┘    └───────────────────────┘  │
│           │                        ▲                 │
│           │              ┌─────────┴─────────┐       │
│           │              │ Environment Vars   │       │
│           │              │ DATABASE_URL ──────┼───┐   │
│           │              │ NODE_ENV=production│   │   │
│           │              └───────────────────┘   │   │
│           │                                      │   │
│           └──────────────────────────────────────┘   │
│                                                     │
└─────────────────────────────────────────────────────┘

           ┌──────────────────────────────────┐
           │ Existing UI-Managed S3 Storage    │
           │       "acme-backups"              │
           │                                   │
           │  ◄── Daily Backup ── PostgreSQL   │
           │       (@daily cron)               │
           └──────────────────────────────────┘
```

## Prerequisites

1. **Running Coolify instance**: a server with Coolify installed and accessible
   via its API endpoint (e.g. `https://coolify.example.com`).
2. **API token**: generate one in Coolify under *Security > API Tokens*.
3. **Server UUID**: the UUID of the destination server registered in Coolify.
   Find it under *Servers* in the Coolify dashboard.
4. *(Optional)* **Existing S3 storage UUID**: only needed if you enable
   backups (`enable_backups = true`). Create the storage in the Coolify web UI
   first, then pass its UUID with `existing_s3_storage_uuid`. The storage can
   target AWS S3, MinIO, Backblaze B2, Cloudflare R2, or another compatible
   backend.

## Usage

Initialize the workspace:

```bash
terraform init
```

Preview the changes:

```bash
terraform plan \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```

Apply:

```bash
terraform apply \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```

To enable daily backups to S3, first create the storage in the Coolify web UI, then pass its UUID here:

```bash
terraform apply \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid" \
  -var="enable_backups=true" \
  -var="existing_s3_storage_uuid=your-s3-storage-uuid"
```

## How It Works

Coolify runs every resource as a Docker container. All resources within the
same project are placed on the **same Docker network**. This means containers
can reach each other using their **resource name as the hostname**, so no need
to expose database ports to the public internet.

In this scenario the `DATABASE_URL` environment variable uses the database
resource's `name` attribute as the hostname:

```
postgresql://acme:PASSWORD@acme-content:5432/acme_content
```

`acme-content` resolves to the PostgreSQL container because Coolify's internal
Docker networking handles DNS within the project network. The database stays
private (`is_public = false`) while the application connects over the internal
network.

## Clean Up

Destroy all resources created by this scenario:

```bash
terraform destroy \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```
