---
name: Bug Report
about: Report a bug in the Coolify Terraform provider
labels: bug
---

## Describe the bug

A clear description of what the bug is.

## Terraform and provider versions

```
terraform version
terraform providers
```

## Coolify version

## Terraform configuration

```hcl
# Paste your relevant .tf configuration here
```

## Expected behavior

## Actual behavior

## Steps to reproduce

1.
2.
3.

## Debug output

Run your command with debug logging enabled and paste the relevant output:

```bash
TF_LOG_PROVIDER=DEBUG terraform plan 2>debug.log
```

**Important:** Review the log for sensitive values (passwords, tokens)
before pasting. The provider redacts most sensitive fields automatically,
but custom environment variable values may appear.

<details>
<summary>Debug log output</summary>

```
Paste relevant log lines here
```

</details>

## Additional context

Any other relevant information (API responses, Coolify UI screenshots, etc.).
