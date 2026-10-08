#!/usr/bin/env python3
"""Fail when Terraform schema rules disagree with the Coolify contract.

Three checks, all local:

1. A field on Hetzner, DigitalOcean, or Vultr createServer that is not on
   ServersController::update_server must use RequiresReplace when the
   provider schema exposes it.
2. coolify_storage_backup has no single-schedule GET. Its schema must not
   use Terraform Default (that fills an import plan and the upsert then
   overwrites the live schedule).
3. Every non-identity attribute of that resource must be listed in
   volumeBackupImportVerifyIgnore. Import cannot read those fields back.

Usage:
    python3 scripts/check-schema-contract.py
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONTRACT = ROOT / "testdata" / "contracts" / "coolify-v4.json"

CREATE_RESOURCES = {
    "HetznerController::createServer": ROOT / "internal/service/hetzner/resource.go",
    "DigitalOceanController::createServer": ROOT / "internal/service/digitalocean/resource.go",
    "VultrController::createServer": ROOT / "internal/service/vultr/resource.go",
}
UPDATE_ENDPOINT = "ServersController::update_server"

VOLUME_BACKUP_RESOURCE = ROOT / "internal/service/volumebackup/resource.go"
VOLUME_BACKUP_ACC = ROOT / "internal/service/volumebackup/resource_acc_test.go"
VOLUME_BACKUP_IGNORE_VAR = "volumeBackupImportVerifyIgnore"
# ImportState sets these. They are compared, not ignored.
VOLUME_BACKUP_IDENTITY = {
    "application_uuid",
    "service_uuid",
    "database_uuid",
    "storage_uuid",
}

ATTR_RE = re.compile(r'"([a-z0-9_]+)":\s*schema\.\w+Attribute\{')
DEFAULT_RE = re.compile(
    r"\b(?:booldefault|int64default|float64default|stringdefault)\."
)


def attribute_bodies(src: str) -> dict[str, str]:
    """Map schema attribute name to the brace-delimited attribute body."""
    bodies: dict[str, str] = {}
    for match in ATTR_RE.finditer(src):
        start = match.end() - 1
        if start < 0 or src[start] != "{":
            continue
        depth = 0
        for i in range(start, len(src)):
            if src[i] == "{":
                depth += 1
            elif src[i] == "}":
                depth -= 1
                if depth == 0:
                    bodies[match.group(1)] = src[start : i + 1]
                    break
    return bodies


def create_only_fields(contract: dict, create_key: str, update_key: str) -> list[str]:
    endpoints = contract.get("endpoints") or {}
    create = set((endpoints.get(create_key) or {}).get("allowed_fields") or [])
    update = set((endpoints.get(update_key) or {}).get("allowed_fields") or [])
    return sorted(create - update)


def missing_requires_replace(schema_src: str, fields: list[str]) -> list[str]:
    blocks = attribute_bodies(schema_src)
    missing = []
    for field in fields:
        body = blocks.get(field)
        if body is None:
            continue
        if "RequiresReplace()" not in body:
            missing.append(field)
    return missing


def schema_attribute_names(schema_src: str) -> list[str]:
    return list(attribute_bodies(schema_src).keys())


def ignore_list(acc_src: str, var_name: str) -> list[str]:
    match = re.search(
        rf"var\s+{re.escape(var_name)}\s*=\s*\[\]string\{{(.*?)\}}",
        acc_src,
        re.DOTALL,
    )
    if not match:
        return []
    return re.findall(r'"([^"]+)"', match.group(1))


def missing_import_ignores(schema_src: str, ignored: list[str], identity: set[str]) -> list[str]:
    ignored_set = set(ignored)
    return [
        name
        for name in schema_attribute_names(schema_src)
        if name not in identity and name not in ignored_set
    ]


def forbidden_schema_defaults(schema_src: str) -> bool:
    return DEFAULT_RE.search(schema_src) is not None


def check_repo(root: Path | None = None, contract_path: Path | None = None) -> list[str]:
    root = root or ROOT
    contract_path = contract_path or (root / "testdata" / "contracts" / "coolify-v4.json")
    problems: list[str] = []
    contract = json.loads(contract_path.read_text())

    for endpoint, rel in CREATE_RESOURCES.items():
        schema_path = rel if root == ROOT else root / rel.relative_to(ROOT)
        if not schema_path.exists():
            problems.append(f"missing schema file for {endpoint}: {schema_path}")
            continue
        src = schema_path.read_text()
        fields = create_only_fields(contract, endpoint, UPDATE_ENDPOINT)
        for field in missing_requires_replace(src, fields):
            problems.append(
                f"{schema_path.relative_to(root)}: {field} is create-only "
                f"({endpoint} not in {UPDATE_ENDPOINT}) and has no RequiresReplace"
            )

    volume_path = VOLUME_BACKUP_RESOURCE if root == ROOT else root / VOLUME_BACKUP_RESOURCE.relative_to(ROOT)
    volume_acc = VOLUME_BACKUP_ACC if root == ROOT else root / VOLUME_BACKUP_ACC.relative_to(ROOT)
    volume_src = volume_path.read_text()
    if forbidden_schema_defaults(volume_src):
        problems.append(
            "internal/service/volumebackup/resource.go uses schema Default. "
            "This resource has no GET. Default fills an import plan and the "
            "upsert overwrites the live schedule. Fill create defaults in Create only."
        )
    acc_src = volume_acc.read_text()
    ignored = ignore_list(acc_src, VOLUME_BACKUP_IGNORE_VAR)
    if not ignored:
        problems.append(
            f"internal/service/volumebackup/resource_acc_test.go is missing {VOLUME_BACKUP_IGNORE_VAR}"
        )
    else:
        for field in missing_import_ignores(volume_src, ignored, VOLUME_BACKUP_IDENTITY):
            problems.append(
                f"{VOLUME_BACKUP_IGNORE_VAR} is missing {field}. "
                "Import cannot read it back (no GET for one schedule)."
            )
    return problems


def main() -> int:
    problems = check_repo()
    if problems:
        print("schema-contract: failed", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        return 1
    print("schema-contract: ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
