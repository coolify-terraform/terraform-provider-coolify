# ACME Corp Custom Docker Compose Stack

This scenario deploys a custom multi-container Docker Compose application
using `docker_compose_raw`, the approach for user-written compose files.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_service` | Custom compose service via `docker_compose_raw` (nginx + redis stack) |
| `data.coolify_service` | Read-back verification via data source |

## How It Works

The `docker_compose_raw` field on `coolify_service` accepts raw Docker Compose
YAML content. This is how you deploy your own compose files through Coolify,
as opposed to using catalog entries (see `acme-integrations` for catalog-based).

This scenario deploys a 2-container stack: nginx serving static content with
a Redis sidecar for caching. In a real project, this YAML would come from your
repository's `docker-compose.yml`.

## Note: git-sourced application compose

This scenario uses **`coolify_service` + inline `docker_compose_raw`**, not a
git-backed `coolify_application` with `build_pack = "dockercompose"`.

For git-sourced compose applications, Coolify only loads `docker_compose_raw`
during a deployment. Setting `docker_compose_domains` before that fails with
`Cannot set docker_compose_domains without docker_compose_raw` on all Coolify
versions supported by this provider (v4.1.0+). Use a two-stage apply (deploy
first, then set domains), or keep using `coolify_service` as shown here.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```
