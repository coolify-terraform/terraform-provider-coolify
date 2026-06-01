# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities through
[GitHub Security Advisories](https://github.com/coolify-terraform/terraform-provider-coolify/security/advisories/new).

Do not open a public issue for security vulnerabilities.

You should receive a response within 48 hours. If the vulnerability is
confirmed, a fix will be released as soon as possible.

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest main | Yes |

## Security Design Principles

This provider follows these security principles:

1. **Minimal privilege**: The provider only requires a Coolify API token. It does
   not need SSH access, database credentials, or host-level permissions.
2. **No credential storage**: API tokens are passed via environment variables or
   provider configuration, never stored in state beyond what Terraform requires.
   Sensitive fields (private keys, passwords, tokens) are marked `Sensitive: true`
   in the schema, which tells Terraform to redact them from plan output.
3. **Input validation at plan time**: UUID format, FQDN, and cron syntax validators
   catch malformed input before any API call is made.
4. **TLS by default**: All API communication uses HTTPS. The HTTP client enforces
   TLS certificate verification.
5. **No shell execution**: The provider never executes shell commands or spawns
   subprocesses. All operations are HTTP API calls.
6. **FIPS 140-3 compliance**: Release builds use `GOFIPS140=latest` for
   FIPS-compliant cryptographic primitives.

## Security Practices

- Dependencies are scanned weekly by Govulncheck, Trivy, and Gitleaks
- Dependabot monitors Go modules and GitHub Actions for updates
- GitHub Dependency Review checks every PR for known vulnerabilities in new dependencies
- FOSSA monitors license compliance
- CodeQL runs on every push and PR for static application security testing (SAST)
- OpenSSF Scorecard runs weekly and reports supply chain security posture
- Test credentials use placeholder values (`"test-token"`)
- No real API tokens or secrets are committed to the repository
- `.gitleaks.toml` allowlists suppress false positives in test fixtures and examples

## Assurance Case

The following measures provide confidence that this provider handles credentials
and infrastructure state correctly:

| Threat | Mitigation | Verification |
|--------|-----------|-------------|
| Credential leakage in logs | All sensitive fields marked `Sensitive: true` | Unit tests verify plan output redaction |
| Credential leakage in state | Terraform encrypts state at rest (user responsibility) | Provider does not control state backend |
| Malicious dependency | Dependabot, Govulncheck, Trivy, Dependency Review | Automated weekly scans + PR-level checks |
| Supply chain attack on CI | Pinned action SHAs, Scorecard monitoring | `scorecard.yml` workflow, no mutable tags |
| API token exposure in examples | Placeholder values enforced by Gitleaks | `.gitleaks.toml` rules + CI enforcement |
| Injection via user input | No shell execution; all input is HTTP API parameters | Static analysis via CodeQL |
