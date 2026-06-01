# Governance

## Overview

terraform-provider-coolify is maintained by a single maintainer with full
commit, release, and administrative access.

## Roles

| Role | Responsibility | Current |
|------|---------------|---------|
| Maintainer | Code review, merge, release, infrastructure | [@SebTardif](https://github.com/SebTardif) |
| Contributor | Submit issues, pull requests, documentation | Anyone |

## Decision Making

Decisions are made by the maintainer. For significant changes (new resources,
breaking API changes, architectural shifts), an issue is opened for discussion
before implementation begins. Contributors are encouraged to open an issue
describing their proposed change before investing time in a pull request.

## Contributions

All contributions are welcome via pull requests. Requirements:

1. All commits must carry a DCO sign-off (`Signed-off-by` trailer)
2. CI must pass (build, lint, test, validate)
3. New features require tests
4. Breaking changes require discussion in an issue first

See [CONTRIBUTING.md](CONTRIBUTING.md) for full development setup and
submission guidelines.

## Releases

Releases follow [Semantic Versioning](https://semver.org/). The release process
is automated via release-please and GoReleaser:

1. Conventional commits on `main` trigger release-please to open a release PR
2. The maintainer merges the release PR
3. GoReleaser builds binaries, signs checksums with GPG, and publishes to GitHub
   Releases and the Terraform Registry

## Access Continuity

The project is hosted under the
[coolify-terraform](https://github.com/coolify-terraform) GitHub organization.
Organization-level settings (branch protection, required CI checks, Dependabot)
ensure that security and quality controls persist regardless of individual
availability.

In the event the maintainer becomes unavailable:

- The repository and all CI/CD infrastructure are under the organization, not a
  personal account
- GPG signing keys for releases are stored in the Terraform Registry namespace
- All automation (CI, Dependabot, release-please) continues to function
  without manual intervention

## Code of Conduct

All participants are expected to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Changes to Governance

This governance model may evolve as the project grows. Changes to this document
follow the same pull request process as code changes.
