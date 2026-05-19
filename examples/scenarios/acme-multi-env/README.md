# ACME Corp Multi-Environment Deployment

## The Story

ACME Corp runs a web application backed by PostgreSQL. The team needs
isolated **dev** and **staging** environments so developers can test without
breaking each other. Both environments deploy the same stack (project,
database, application, and environment variables) but with different
configuration: a smaller database image in dev, different Git branches,
and separate project names so Coolify keeps everything neatly separated.

Instead of copy-pasting Terraform resources, ACME wraps the stack in a
reusable **Terraform module** and calls it twice, once per environment.

## What Gets Created

Each module call provisions:

| Resource | Description |
|---|---|
| `coolify_project` | A Coolify project that groups the environment's resources |
| `coolify_database_postgresql` | A PostgreSQL database for the app |
| `coolify_application` | The web application deployed from Git via Nixpacks |
| `coolify_environment_variable` | `DATABASE_URL` connecting the app to its database |

Two module calls (`dev` + `staging`) means **8 resources total**.

```
┌──────────────────────────────────┐  ┌──────────────────────────────────┐
│          module "dev"            │  │        module "staging"          │
│                                  │  │                                  │
│  ┌────────────────────────────┐  │  │  ┌────────────────────────────┐  │
│  │   coolify_project          │  │  │  │   coolify_project          │  │
│  │   "acme-dev"               │  │  │  │   "acme-staging"           │  │
│  └────────────────────────────┘  │  │  └────────────────────────────┘  │
│                                  │  │                                  │
│  ┌────────────────────────────┐  │  │  ┌────────────────────────────┐  │
│  │   coolify_postgresql_db    │  │  │  │   coolify_postgresql_db    │  │
│  │   db: acme_dev             │  │  │  │   db: acme_staging         │  │
│  └──────────┬─────────────────┘  │  │  └──────────┬─────────────────┘  │
│             │ DATABASE_URL       │  │             │ DATABASE_URL       │
│  ┌──────────▼─────────────────┐  │  │  ┌──────────▼─────────────────┐  │
│  │   coolify_application      │  │  │  │   coolify_application      │  │
│  │   branch: main             │  │  │  │   branch: main             │  │
│  └────────────────────────────┘  │  │  └────────────────────────────┘  │
└──────────────────────────────────┘  └──────────────────────────────────┘
```

## Prerequisites

- A running [Coolify](https://coolify.io/) instance with API access enabled.
- A server already registered in Coolify (you need its UUID).
- An API token generated from the Coolify dashboard.
- Terraform ≥ 1.0 installed.

## Usage

```bash
# Initialize the workspace
terraform init

# Preview what will be created
terraform plan \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=YOUR_API_TOKEN" \
  -var="server_uuid=YOUR_SERVER_UUID"

# Apply to create both environments
terraform apply \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=YOUR_API_TOKEN" \
  -var="server_uuid=YOUR_SERVER_UUID"
```

Or create a `terraform.tfvars` file:

```hcl
coolify_endpoint = "https://coolify.example.com"
coolify_token    = "YOUR_API_TOKEN"
server_uuid      = "YOUR_SERVER_UUID"
```

Then simply run:

```bash
terraform init
terraform apply
```

## How It Works

1. **`modules/stack/`** defines a self-contained Coolify stack: project →
   database → application → environment variable. Every resource name and
   database is parameterized so the module can be called multiple times
   without conflicts.

2. **`main.tf`** (root) calls the module twice (`module "dev"` and
   `module "staging"`), each with its own project name, database name, and
   Git branch. Both share the same `server_uuid` so they deploy to the
   same Coolify server.

3. Terraform provisions both stacks in parallel (where the dependency
   graph allows), creating two fully isolated environments from one
   codebase.

## Clean Up

Destroy all resources in both environments:

```bash
terraform destroy \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=YOUR_API_TOKEN" \
  -var="server_uuid=YOUR_SERVER_UUID"
```

## Key Concepts

- **Module composition**: Wrap a set of Coolify resources into a Terraform
  module, then instantiate it multiple times with different variables. This
  is the standard Terraform pattern for managing multiple environments
  without duplicating configuration.

- **Implicit wiring**: The `DATABASE_URL` environment variable is built
  from the database resource's attributes. Terraform handles the
  dependency ordering automatically; the database is created before the
  environment variable that references it.

- **Environment isolation**: Each module call creates its own Coolify
  project, so resources never collide. You can add a `production` module
  call later without touching the existing environments.
