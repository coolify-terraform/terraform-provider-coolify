#!/usr/bin/env python3
"""Check endpoint field compatibility across Coolify contract versions.

Compares the allowed_fields in each endpoint across all versioned contract
files (testdata/contracts/coolify-v4.*.json, excluding coolify-v4.json and
coolify-v4-latest.json which are aliases). Reports fields that exist in some
versions but not others, helping identify version-dependent features that
need UseStateForUnknown() instead of Default in the provider schema.

Usage:
    python3 scripts/check-contract-compat.py [--ci] [--all]

With --ci, exits non-zero if any version-dependent fields are found that
are not listed in the known exceptions file. Without --ci, prints a report
only.

With --all, compares across all contract versions. Without --all (default),
only compares the two most recent versions, which is the useful check for
catching regressions between adjacent releases.
"""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONTRACTS_DIR = ROOT / "testdata" / "contracts"

# Fields we know are version-dependent and have been handled correctly
# in the provider (UseStateForUnknown, documented minimum version, etc.).
# Add entries here when a field is intentionally version-dependent.
KNOWN_VERSION_DEPENDENT: dict[str, set[str]] = {
    "DatabasesController::update_by_uuid": {
        "health_check_enabled",
        "health_check_interval",
        "health_check_timeout",
        "health_check_retries",
        "health_check_start_period",
    },
}


def _parse_version(name: str) -> tuple[int, ...]:
    """Extract a sortable version tuple from a filename like coolify-v4.1.2."""
    m = re.search(r"v(\d+(?:\.\d+)*)", name)
    if not m:
        return (0,)
    return tuple(int(x) for x in m.group(1).split("."))


def load_contracts() -> dict[str, dict]:
    """Load all versioned contract files, excluding aliases."""
    contracts = {}
    for path in sorted(CONTRACTS_DIR.glob("coolify-v4*.json")):
        # Skip the unversioned alias and the latest alias
        if path.name in ("coolify-v4.json", "coolify-v4-latest.json"):
            continue
        contracts[path.stem] = json.loads(path.read_text())
    return dict(sorted(contracts.items(), key=lambda kv: _parse_version(kv[0])))


def compare_endpoints(contracts: dict[str, dict]) -> list[dict]:
    """Compare allowed_fields across versions for each endpoint."""
    # Collect all endpoints across all versions
    all_endpoints: set[str] = set()
    for data in contracts.values():
        all_endpoints.update(data.get("endpoints", {}).keys())

    results = []
    for endpoint in sorted(all_endpoints):
        # Gather fields per version
        version_fields: dict[str, set[str]] = {}
        for version, data in contracts.items():
            ep = data.get("endpoints", {}).get(endpoint)
            if ep is not None:
                version_fields[version] = set(ep.get("allowed_fields", []))

        if len(version_fields) < 2:
            continue

        # Find the union and intersection
        all_fields = set().union(*version_fields.values())
        common_fields = set.intersection(*version_fields.values())
        version_dependent = all_fields - common_fields

        if version_dependent:
            # For each version-dependent field, find which versions have it
            field_details = []
            for field in sorted(version_dependent):
                present_in = sorted(
                    [v for v, fields in version_fields.items() if field in fields],
                    key=_parse_version,
                )
                absent_from = sorted(
                    [v for v, fields in version_fields.items() if field not in fields],
                    key=_parse_version,
                )
                field_details.append({
                    "field": field,
                    "present_in": present_in,
                    "absent_from": absent_from,
                    "min_version": present_in[0] if present_in else "unknown",
                })

            results.append({
                "endpoint": endpoint,
                "total_fields": len(all_fields),
                "common_fields": len(common_fields),
                "version_dependent": field_details,
            })

    return results


def main():
    ci_mode = "--ci" in sys.argv
    all_mode = "--all" in sys.argv

    contracts = load_contracts()
    if not all_mode and len(contracts) > 2:
        # Keep only the two most recent versions
        keys = list(contracts.keys())
        contracts = {k: contracts[k] for k in keys[-2:]}
    if len(contracts) < 2:
        print("Need at least 2 versioned contracts to compare.")
        print(f"Found: {list(contracts.keys())}")
        sys.exit(0)

    print(f"Comparing {len(contracts)} contract versions: {', '.join(contracts.keys())}")
    print()

    results = compare_endpoints(contracts)

    if not results:
        print("All endpoint fields are consistent across all versions.")
        sys.exit(0)

    unknown_count = 0
    for r in results:
        endpoint = r["endpoint"]
        known = KNOWN_VERSION_DEPENDENT.get(endpoint, set())
        unknown_fields = [
            f for f in r["version_dependent"] if f["field"] not in known
        ]
        known_fields = [
            f for f in r["version_dependent"] if f["field"] in known
        ]

        print(f"## {endpoint}")
        print(f"   Total fields (union): {r['total_fields']}, common to all versions: {r['common_fields']}")

        if known_fields:
            print(f"   Known version-dependent (handled in provider):")
            for f in known_fields:
                print(f"     {f['field']}: added in {f['min_version']}, absent from {', '.join(f['absent_from'])}")

        if unknown_fields:
            print(f"   NEW version-dependent (needs provider fix):")
            for f in unknown_fields:
                print(f"     {f['field']}: added in {f['min_version']}, absent from {', '.join(f['absent_from'])}")
            unknown_count += len(unknown_fields)

        print()

    if unknown_count > 0:
        print(f"Found {unknown_count} version-dependent field(s) not in the known exceptions list.")
        print("For each field, either:")
        print("  1. Fix the provider (UseStateForUnknown instead of Default, document min version)")
        print("  2. Add to KNOWN_VERSION_DEPENDENT in scripts/check-contract-compat.py")
        if ci_mode:
            sys.exit(1)
    else:
        print("All version-dependent fields are accounted for.")


if __name__ == "__main__":
    main()