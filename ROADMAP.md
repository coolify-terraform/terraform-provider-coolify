# Roadmap

This document outlines the planned direction for terraform-provider-coolify.
Priorities may shift based on community feedback and upstream Coolify API changes.

## Current Status (v0.1.x)

The provider covers the core Coolify resource model:

- 33 managed resources (projects, servers, applications, databases, services,
  backups, environment variables, deployments, and more)
- 44 data sources for reading existing infrastructure
- 17 ACME Corp scenario examples with integration tests
- Full import support for adopting existing Coolify resources

## Near Term

### OpenTofu Registry publication

Register the provider on the OpenTofu Registry for users who use OpenTofu
instead of HashiCorp Terraform. ([#414](https://github.com/coolify-terraform/terraform-provider-coolify/issues/414))

### Notification channel resources

Add resources for managing Coolify notification channels (Slack, Discord, email,
Telegram, etc.) once the upstream API supports full CRUD.
([#394](https://github.com/coolify-terraform/terraform-provider-coolify/issues/394),
blocked on upstream)

### S3 storage resource

Add a `coolify_s3_storage` resource for managing S3-compatible backup
destinations once the upstream API exposes top-level S3 storage endpoints.
([#393](https://github.com/coolify-terraform/terraform-provider-coolify/issues/393),
blocked on upstream)

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
