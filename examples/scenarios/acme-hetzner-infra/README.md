# ACME Corp Hetzner Cloud Infrastructure

This scenario provisions Hetzner Cloud servers through Coolify, demonstrating
the full workflow for cloud-based infrastructure:

1. **SSH deploy key** (`coolify_private_key`) for authenticating to
   provisioned servers.
2. **Hetzner cloud token** (`coolify_cloud_token`) to register the Hetzner
   API token with Coolify.
3. **Production server** (`coolify_server_hetzner`) -- a cx22 instance in
   Falkenstein for running applications.
4. **Build server** (`coolify_server_hetzner`) -- a cx32 instance in
   Nuremberg dedicated to builds.

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform apply
```

> **Note:** This scenario provisions real Hetzner Cloud servers and incurs
> costs. The included test uses `command = plan` to validate configuration
> without creating resources.

## Resources Created

| Resource | Purpose |
|----------|---------|
| `coolify_private_key.deploy` | SSH key for server access |
| `coolify_cloud_token.hetzner` | Hetzner API token registration |
| `coolify_server_hetzner.production` | Production application server (cx22, fsn1) |
| `coolify_server_hetzner.build` | Dedicated build server (cx32, nbg1) |
