---
page_title: "Architecture"
subcategory: "Development"
description: |-
  Internal architecture of the Coolify Terraform provider.
---

# Architecture

This guide describes the internal structure of the provider for contributors
and maintainers. End users do not need to read this; see the
[Quick Start](quickstart) instead.

## Layer diagram

```
┌─────────────────────────────────────────────────┐
│  Terraform CLI / Plugin Protocol (gRPC)         │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  Provider (internal/provider/)                   │
│  - Schema: endpoint + token configuration        │
│  - Configure: health check via GET /api/v1/ver.  │
│  - Registration: lists all resources + data src  │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  Service Layer (internal/service/<type>/)         │
│  - One subpackage per resource type              │
│  - resource.go: CRUD + ImportState               │
│  - data_source.go: Read-only                     │
│  - Expand (HCL -> API) / Flatten (API -> HCL)   │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  Client (internal/client/)                       │
│  - One file per resource type                    │
│  - retryablehttp with 3 retries, 30s timeout     │
│  - NotFoundError + IsNotFound() helper           │
│  - context.Context on every method               │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  Coolify API (HTTPS)                             │
└─────────────────────────────────────────────────┘
```

## Package responsibilities

| Package | Purpose |
|---------|---------|
| `internal/provider/` | Provider schema, `Configure` (creates the HTTP client), resource and data source registration |
| `internal/service/<type>/` | One subpackage per Coolify resource type. Contains the resource, data source(s), expand/flatten helpers, and unit tests |
| `internal/client/` | Typed HTTP client. One file per resource type with Create/Get/Update/Delete/List methods |
| `internal/flex/` | Type conversion between Go native types and Terraform Plugin Framework types (`types.String`, `types.Int64`, etc.) |
| `internal/acctest/` | Shared test utilities: provider factories, mock server wrappers (`WithVersionEndpoint`), acceptance test helpers |
| `internal/validate/` | Custom validators: UUID format, FQDN URL, cron syntax |
| `internal/filter/` | Generic filtering helpers for plural data sources |
| `internal/spectest/` | OpenAPI spec compliance tests, API coverage tracking, contract coverage verification |
| `scripts/` | Python tooling: contract extraction from Coolify source, OpenAPI spec generation, contract diffing |
| `testdata/contracts/` | Versioned contract JSON extracted from Coolify PHP source (source of truth for field definitions) |

## Request lifecycle

A typical `terraform apply` for a resource follows this path:

1. **Terraform CLI** calls the provider's gRPC `ApplyResourceChange` endpoint
2. **Provider framework** routes to the resource's `Create`, `Update`, or `Delete` method
3. **Service layer** expands the Terraform plan into an API request struct
4. **Client** sends the HTTP request to Coolify with retries and timeout
5. **Client** parses the response, returns a typed struct or `NotFoundError`
6. **Service layer** flattens the API response back into Terraform state
7. **Provider framework** returns the new state to Terraform CLI

## Expand / Flatten pattern

Every resource has two directional conversions:

- **Expand**: Reads Terraform plan attributes (`types.String`, `types.Bool`) and
  builds the Go struct that the client sends to the API. Runs during Create and
  Update.
- **Flatten**: Takes the API response struct and writes values back into the
  Terraform state model. Runs during Create (read-back), Read, and Import.

Sensitive fields (passwords, private keys) require special flatten handling:
when the API returns an empty string for a sensitive field, the flatten function
preserves the value from Terraform state rather than overwriting it.

## Testing architecture

```
┌─────────────────────────────────────────┐
│  Unit Tests (make test)                  │
│  - httptest mock servers per resource    │
│  - resource.UnitTest wrapper             │
│  - No real Coolify instance needed       │
│  - Race detector enabled (-race)         │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  Acceptance Tests (make testacc)         │
│  - Real Coolify instance (TF_ACC=1)     │
│  - resource.Test wrapper                 │
│  - Serialized execution (no parallelism) │
│  - Full CRUD + import verification       │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  Scenario Tests (terraform test)         │
│  - .tftest.hcl files in examples/scen.   │
│  - Multi-resource integration tests      │
│  - Real Coolify instance                 │
│  - 16 ACME Corp scenarios                │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  Spec Compliance (make spec-check)       │
│  - Contract coverage verification        │
│  - OpenAPI spec drift detection          │
│  - API endpoint coverage tracking        │
└─────────────────────────────────────────┘
```

## Contract extraction pipeline

The provider uses a contract-first approach anchored to the Coolify PHP source
code (not the OpenAPI spec):

1. `make contract-extract` clones the Coolify repo and runs
   `scripts/extract-contract.py` to parse PHP models, controllers, and
   migrations into `testdata/contracts/coolify-v4.json`
2. `make contract-check` verifies that Go client structs cover all contract
   fields
3. `make spec-generate` produces an OpenAPI spec from the contract
4. `make spec-check` runs the spec compliance test suite

This pipeline catches API drift automatically when Coolify updates its models.

## CI pipeline

The CI pipeline runs 9 jobs on every push and PR:

| Job | What it checks |
|-----|---------------|
| Detect Changes | Path-based filtering to skip unchanged jobs |
| DCO | Signed-off-by trailer on all PR commits |
| Test | Unit tests with race detector + code coverage |
| Lint | golangci-lint (20 linters) + Govulncheck + GoReleaser check |
| Validate | HCL formatting + tfplugindocs + Trivy + Gitleaks |
| Acceptance Tests | Full suite against a bootstrapped Coolify instance |
| Scenario Tests | `terraform test` against a real Coolify instance |
| Contract Freshness | Weekly check that the contract matches upstream |
| CI | Gate job that requires all other jobs to pass |
