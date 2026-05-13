---
page_title: "Installation"
subcategory: "Getting Started"
description: |-
  Install and configure the Coolify Terraform provider.
---

# Installation

## Prerequisites

| Requirement | Minimum Version |
|-------------|-----------------|
| Terraform   | 1.0+            |
| Coolify     | v4.x            |

## Install from Terraform Registry

Add the provider to your `required_providers` block:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}
```

Run `terraform init` to download and install the provider:

```bash
terraform init
```

## Configure authentication

Generate an API token in the Coolify UI: **Security > API Tokens**.

### Token permissions

For full functionality, create a token with **root** permission. Without
`root` or `read:sensitive`, the Coolify API hides sensitive fields
(Dockerfiles, compose files, private keys, database passwords) from API
responses. This causes empty state values and unexpected diffs during
`terraform plan` and `terraform import`.

| Permission | What it unlocks |
|------------|-----------------|
| `root` | Full access to all fields (recommended for Terraform) |
| `read:sensitive` | Read-only access to sensitive fields |
| Default | Sensitive fields are hidden from responses |

Set the credentials as environment variables (recommended):

```bash
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"
```

Or configure them directly in the provider block:

```hcl
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = "your-api-token"
}
```

## Verify the connection

Create a file called `main.tf`:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}

data "coolify_version" "current" {}

output "coolify_version" {
  value = data.coolify_version.current.version
}
```

```bash
terraform init
terraform apply
```

If the connection is successful, the output shows your Coolify instance version.

## Install from source

```bash
git clone https://github.com/SebTardifLabs/terraform-provider-coolify.git
cd terraform-provider-coolify
make install
```

This compiles the provider and places it in your local Terraform plugin
cache.

## Next steps

Read [Core Concepts](concepts) to understand the resource model, then
follow the [Quick Start](quickstart) to deploy your first application.
