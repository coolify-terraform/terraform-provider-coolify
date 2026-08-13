# Roadmap

This document outlines the planned direction for terraform-provider-coolify.
Priorities may shift based on community feedback and upstream Coolify API changes.

## Current Status (v0.1.x)

The provider covers the core Coolify resource model:

- 45 managed resources (projects, servers, applications, databases, services,
  backups, environment variables, deployments, notification channels, S3 storage,
  and more)
- 56 data sources for reading existing infrastructure
- 17 ACME Corp scenario examples with integration tests
- Full import support for adopting existing Coolify resources
- Coolify v4.2 surfaces: destinations, DigitalOcean/Vultr server provisioning
- Coolify v4.3 surfaces: notification channels, S3 storage, docker/compose version probes
- Published on both [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify)
  and [OpenTofu Registry](https://search.opentofu.org/provider/coolify-terraform/coolify)
  ([#414](https://github.com/coolify-terraform/terraform-provider-coolify/issues/414))

## Near Term

### ~~Notification channel resources~~ (shipped)

Team-scoped notification resources ship for Coolify >= v4.3.0 (provider v0.1.14+):
email, Discord, Slack, Telegram, Pushover, and generic webhook
([#394](https://github.com/coolify-terraform/terraform-provider-coolify/issues/394),
[#704](https://github.com/coolify-terraform/terraform-provider-coolify/issues/704)).

### ~~S3 storage resource~~ (shipped)

`coolify_s3_storage` and `coolify_s3_storages` ship for Coolify >= v4.3.0
(provider v0.1.14+). See the [version support guide](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Medium Term

### Upstream API parity

As Coolify v4 stabilizes its API, update the provider to track new endpoints
and fields. The contract extraction pipeline (`make contract-extract`) and spec
compliance tests (`make spec-check`) automate detection of API drift.

### Additional scenario examples

Expand the ACME Corp scenario library with more real-world deployment patterns
(multi-server setups, database clustering, blue-green deployments).

## Long Term

### Coolify v5 support

When Coolify v5 is released, evaluate API changes and plan a migration path.
Breaking changes will follow semantic versioning with a deprecation period.

## Contributing

Feature requests and priority feedback are welcome via
[GitHub Issues](https://github.com/coolify-terraform/terraform-provider-coolify/issues).
If a feature you need is missing, open an issue describing your use case.
