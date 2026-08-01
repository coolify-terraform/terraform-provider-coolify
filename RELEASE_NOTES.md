## Highlights

v0.1.11 lets you create internal applications without a public Traefik host,
and surfaces more Coolify application and storage settings that were previously
writeable only outside Terraform. Use `autogenerate_domain = false` when an app
must not get an sslip or wildcard FQDN.

## New features

- **`autogenerate_domain` on all application resources.** Coolify defaults this
  to `true` on create and generates a public FQDN when `domains` is blank
  (`https://{uuid}.{wildcard}` or `http://{uuid}.{ip}.sslip.io`). The provider
  now sends the flag (default `true`). Set `false` for workers, queues, and
  other internal apps so no public host is installed. Guides and examples cover
  the internal-app pattern ([#646](https://github.com/coolify-terraform/terraform-provider-coolify/pull/646), [#648](https://github.com/coolify-terraform/terraform-provider-coolify/pull/648)).

```hcl
resource "coolify_application_docker_image" "worker" {
  name                = "background-worker"
  project_uuid        = coolify_project.main.uuid
  server_uuid         = data.coolify_server.prod.uuid
  docker_image        = "myorg/worker:latest"
  ports_exposes       = "8080"
  autogenerate_domain = false
}
```

- **More application settings and storage preview flag.** Deferred contract
  write fields are now managed in Terraform, including git submodule/LFS/shallow
  options, build-cache and compose-related settings, env sorting, preview-related
  toggles, and storage `is_preview_suffix_enabled` (create uses a post-create PATCH
  when the API rejects the field on POST) ([#641](https://github.com/coolify-terraform/terraform-provider-coolify/pull/641)).

- **Empty `domains` in config.** The validator allows `domains = ""` so plans can
  express clear intent. Clearing an **existing** FQDN on update still depends on
  Coolify accepting empty `domains` on PATCH (tracked as
  [#647](https://github.com/coolify-terraform/terraform-provider-coolify/issues/647);
  UI-side clear was fixed upstream separately). Prefer
  `autogenerate_domain = false` for new internal apps.

## Bug fixes

- **Storage create could 422 when sending `is_preview_suffix_enabled`.** Create
  no longer sends that field on POST; when the desired value is non-default,
  a follow-up PATCH applies it so first apply converges ([#641](https://github.com/coolify-terraform/terraform-provider-coolify/pull/641)).

## Compatibility

| Surface | Requirement |
|---------|-------------|
| Existing resources | Still work on Coolify **>= 4.1.0** |
| `autogenerate_domain` | All supported Coolify versions (create allow-list) |
| Settings from #641 | Where your Coolify API exposes them (see attribute docs) |

No Terraform state migration is required. Pin `version = "~> 0.1.11"` (or
newer) when you want the new attributes.

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.11"
    }
  }
}
```

Then run `terraform init -upgrade` and review the plan for new optional
attributes (defaults match Coolify).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.10...v0.1.11
