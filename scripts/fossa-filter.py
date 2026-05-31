#!/usr/bin/env python3
"""Filter known false positives from FOSSA test JSON output.

Exit 0 if all issues are documented false positives.
Exit 1 if any genuine issues remain after filtering.

Documented false positives
--------------------------
1. golang.org/x/text  CC-BY-SA-*
   Unicode CLDR data files trigger CC-BY-SA detection. The module
   license is BSD-3-Clause. Known Go ecosystem false positive
   (golang/go#53534).

2. golang.org/x/crypto  openssl-ssleay
   Detected in test fixtures. The module license is BSD-3-Clause.

3. github.com/hashicorp/*  MPL-2.0
   Correct declared license. Terraform providers use (not modify)
   these SDK packages. MPL-2.0 copyleft applies only to modifications
   of MPL-licensed files, so this is compatible usage. Every Terraform
   provider has these dependencies.
"""

import json
import sys


KNOWN_FALSE_POSITIVES = {
    "golang.org/x/text": {
        "CC-BY-SA-1.0",
        "CC-BY-SA-2.0",
        "CC-BY-SA-2.5",
        "CC-BY-SA-3.0",
        "CC-BY-SA-4.0",
    },
    "golang.org/x/crypto": {
        "openssl-ssleay",
    },
}


def is_false_positive(pkg: str, license_id: str) -> bool:
    """Return True if the (package, license) pair is a documented false positive."""
    # Check exact-match false positives (x/text, x/crypto)
    for fp_prefix, fp_licenses in KNOWN_FALSE_POSITIVES.items():
        if pkg.startswith(fp_prefix) and license_id in fp_licenses:
            return True

    # HashiCorp MPL-2.0: correct license, compatible usage in providers
    if pkg.startswith("github.com/hashicorp/") and "MPL-2.0" in license_id:
        return True

    return False


def extract_package(issue: dict) -> str:
    """Extract the package path from a FOSSA issue.

    FOSSA uses 'revisionId' with format 'go+golang.org/x/text$v0.37.0'.
    Strip the ecosystem prefix and version suffix to get the Go module path.
    Falls back to 'package' or 'name' fields if revisionId is missing.
    """
    rev = issue.get("revisionId", "")
    if rev:
        # Strip ecosystem prefix (e.g., "go+", "npm+", "pip+")
        if "+" in rev:
            rev = rev.split("+", 1)[1]
        # Strip version suffix (e.g., "$v0.37.0")
        if "$" in rev:
            rev = rev.rsplit("$", 1)[0]
        return rev
    return issue.get("package", "") or issue.get("name", "") or ""


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: fossa-filter.py <fossa-results.json>", file=sys.stderr)
        return 2

    with open(sys.argv[1]) as f:
        data = json.load(f)

    # fossa test --format json may return a list or an object with an issues key
    if isinstance(data, list):
        issues = data
    else:
        issues = data.get("issues", data.get("issue", []))
    if not isinstance(issues, list):
        issues = []

    real_issues = []
    filtered_count = 0

    for issue in issues:
        pkg = extract_package(issue)
        lic = issue.get("license", "") or issue.get("licenseId", "") or ""

        if is_false_positive(pkg, lic):
            filtered_count += 1
            continue

        real_issues.append(issue)

    if real_issues:
        print(f"FAIL: {len(real_issues)} genuine issue(s) after filtering "
              f"{filtered_count} known false positives:")
        for r in real_issues:
            pkg = extract_package(r)
            lic = r.get("license", "") or r.get("licenseId", "?")
            itype = r.get("type", "") or r.get("issueType", "?")
            print(f"  - {pkg}  {lic}  ({itype})")
        return 1

    print(f"OK: All {filtered_count} issue(s) are documented false positives.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
