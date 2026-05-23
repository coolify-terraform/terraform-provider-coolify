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

## Security Practices

- Dependencies are scanned weekly by Govulncheck, Trivy, and Gitleaks
- Dependabot monitors Go modules and GitHub Actions for updates
- Test credentials use placeholder values (`"test-token"`)
- No real API tokens or secrets are committed to the repository
