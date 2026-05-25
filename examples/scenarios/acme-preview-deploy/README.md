# ACME Corp Preview Deployments

This scenario sets up PR-based preview environments using a GitHub App
integration, demonstrating the full workflow from app registration to
per-PR preview deployments:

1. **GitHub App private key** (`coolify_private_key`) for authenticating
   the GitHub App with Coolify.
2. **GitHub App integration** (`coolify_github_app`) registered with
   Coolify using your app credentials.
3. **Application via GitHub App** (`coolify_application_github_app`)
   deployed from a GitHub repository through the registered app.
4. **Preview environments** (`coolify_application_preview`) created for
   specific pull requests -- cleaned up automatically on `terraform destroy`.

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform apply
```

## Resources Created

| Resource | Purpose |
|----------|---------|
| `coolify_project.acme` | Project container |
| `coolify_private_key.github_app` | PEM key for GitHub App auth |
| `coolify_github_app.acme` | GitHub App integration |
| `coolify_application_github_app.web` | Application deployed via GitHub App |
| `coolify_application_preview.pr_1` | Preview environment for PR #1 |
| `coolify_application_preview.pr_2` | Preview environment for PR #2 |

## Plan-Only Testing

This scenario requires real GitHub App credentials to apply. The included
test file validates HCL syntax, the resource graph, and schema correctness
using `command = plan` with fake credentials.
