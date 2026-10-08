#!/usr/bin/env python3
"""Fail when Terraform schema rules disagree with the Coolify contract.

Checks, all local:

1. Create-only fields (on the create allow-list, absent from the matching
   update allow-list) must use a replace plan modifier when the schema
   exposes them. A description that says the change does not force
   replacement is exempt (the provider keeps state and does not send an
   update).
2. Schema Default combined with unconditional RequiresReplace() must be
   copied back when Read sees null or unknown, and the copy must be the
   same constant. Otherwise import plans a destroy/create.
3. coolify_storage_backup has no single-schedule GET. Its schema must not
   use Terraform Default. Every non-identity attribute must be listed in
   volumeBackupImportVerifyIgnore.

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

# create endpoint, update endpoint, schema directories or files (not tests,
# not data sources).
CREATE_SURFACES = (
    ("HetznerController::createServer", "ServersController::update_server", ("internal/service/hetzner",)),
    ("DigitalOceanController::createServer", "ServersController::update_server", ("internal/service/digitalocean",)),
    ("VultrController::createServer", "ServersController::update_server", ("internal/service/vultr",)),
    ("GithubController::create_github_app", "GithubController::update_github_app", ("internal/service/githubapp",)),
    ("GitlabController::create_gitlab_app", "GitlabController::update_gitlab_app", ("internal/service/gitlabapp",)),
    ("ApplicationsController::create_application", "ApplicationsController::update_by_uuid", ("internal/service/application",)),
    ("ServicesController::create_service", "ServicesController::update_by_uuid", ("internal/service/service",)),
    ("DatabasesController::create_database", "DatabasesController::update_by_uuid", ("internal/service/database/common.go",)),
    ("DestinationsController::create", "DestinationsController::update", ("internal/service/destination",)),
    ("CloudProviderTokensController::store", "CloudProviderTokensController::update", ("internal/service/cloudtoken",)),
)
REPLACE_TOKENS = (
    "RequiresReplace()",
    "RequiresReplaceIfConfigured()",
    "RequiresReplaceIf(",
    "RequiresReplaceIfKnown()",
    "EnvironmentNamePlan()",
)
NO_REPLACE_PHRASE = "does not force replacement"

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


def has_replace_modifier(body: str) -> bool:
    if NO_REPLACE_PHRASE in body:
        return True
    return any(token in body for token in REPLACE_TOKENS)


def missing_requires_replace(schema_src: str, fields: list[str]) -> list[str]:
    blocks = attribute_bodies(schema_src)
    missing = []
    for field in fields:
        body = blocks.get(field)
        if body is None:
            continue
        if not has_replace_modifier(body):
            missing.append(field)
    return missing


def static_default(body: str) -> tuple[str, str] | None:
    """Return (kind, literal) for a schema Default constant, if any."""
    match = re.search(r"booldefault\.StaticBool\((true|false)\)", body)
    if match:
        return ("bool", match.group(1))
    match = re.search(r"int64default\.StaticInt64\((-?\d+)\)", body)
    if match:
        return ("int64", match.group(1))
    match = re.search(r"float64default\.StaticFloat64\((-?\d+(?:\.\d+)?)\)", body)
    if match:
        return ("float64", match.group(1))
    match = re.search(r'stringdefault\.StaticString\("([^"]*)"\)', body)
    if match:
        return ("string", match.group(1))
    return None


def go_field_name(package_src: str, attr: str) -> str | None:
    match = re.search(
        rf"(\w+)\s+types\.\w+\s+`tfsdk:\"{re.escape(attr)}\"`",
        package_src,
    )
    if not match:
        return None
    return match.group(1)


def fill_assignment(kind: str, go_name: str, value: str) -> str:
    if kind == "bool":
        return f"{go_name} = types.BoolValue({value})"
    if kind == "int64":
        return f"{go_name} = types.Int64Value({value})"
    if kind == "float64":
        return f"{go_name} = types.Float64Value({value})"
    return f'{go_name} = types.StringValue("{value}")'


def default_replace_gaps(schema_src: str, package_src: str) -> list[str]:
    """Default plus RequiresReplace() must be restored on read when null."""
    gaps = []
    for name, body in attribute_bodies(schema_src).items():
        if "RequiresReplace()" not in body:
            continue
        default = static_default(body)
        if default is None:
            continue
        kind, value = default
        go_name = go_field_name(package_src, name)
        if go_name is None:
            gaps.append(f"{name}: Default and RequiresReplace() but no tfsdk field")
            continue
        assignment = fill_assignment(kind, go_name, value)
        restores_null = "IsNull()" in package_src or "IsUnknown()" in package_src
        if assignment not in package_src or not restores_null:
            gaps.append(
                f"{name}: Default {value} plus RequiresReplace() is not copied "
                f"when Read sees null ({assignment})"
            )
    return gaps


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


def description_string_to_framework_lines(src: str) -> list[int]:
    """Line numbers where a non-comment assigns Description via StringToFramework."""
    hits = []
    for lineno, line in enumerate(src.splitlines(), 1):
        stripped = line.lstrip()
        if stripped.startswith("//"):
            continue
        if "StringToFramework" in line and "Description" in line:
            hits.append(lineno)
    return hits


def description_string_to_framework(root: Path) -> list[str]:
    """Resource flattens must not turn a configured empty description into null."""
    problems = []
    service = root / "internal" / "service"
    if not service.is_dir():
        return problems
    for path in sorted(service.rglob("*.go")):
        name = path.name
        if name.endswith("_test.go") or "data_source" in name:
            continue
        for lineno in description_string_to_framework_lines(path.read_text()):
            problems.append(
                f"{path.relative_to(root)}:{lineno}: Description uses StringToFramework. "
                "Use StringFromAPI so a configured empty string stays empty."
            )
    return problems


def forbidden_schema_defaults(schema_src: str) -> bool:
    return DEFAULT_RE.search(schema_src) is not None


def schema_files(root: Path, rels: tuple[str, ...]) -> list[Path]:
    files: list[Path] = []
    for rel in rels:
        path = root / rel
        if path.is_file():
            files.append(path)
            continue
        if not path.is_dir():
            continue
        for candidate in sorted(path.rglob("*.go")):
            name = candidate.name
            if name.endswith("_test.go") or "data_source" in name:
                continue
            files.append(candidate)
    return files


def check_repo(root: Path | None = None, contract_path: Path | None = None) -> list[str]:
    root = root or ROOT
    contract_path = contract_path or (root / "testdata" / "contracts" / "coolify-v4.json")
    problems: list[str] = []
    contract = json.loads(contract_path.read_text())

    for create_key, update_key, rels in CREATE_SURFACES:
        files = schema_files(root, rels)
        if not files:
            problems.append(f"missing schema files for {create_key}: {rels}")
            continue
        src = "\n".join(path.read_text() for path in files)
        fields = create_only_fields(contract, create_key, update_key)
        for field in missing_requires_replace(src, fields):
            problems.append(
                f"{rels[0]}: {field} is create-only "
                f"({create_key} not in {update_key}) and has no replace modifier"
            )

    service_root = root / "internal" / "service"
    if service_root.is_dir():
        seen: set[str] = set()
        for path in sorted(service_root.rglob("*.go")):
            name = path.name
            if name.endswith("_test.go") or "data_source" in name:
                continue
            package_dir = path.parent
            key = str(package_dir)
            if key in seen:
                continue
            seen.add(key)
            package_files = [
                item
                for item in sorted(package_dir.glob("*.go"))
                if not item.name.endswith("_test.go") and "data_source" not in item.name
            ]
            package_src = "\n".join(item.read_text() for item in package_files)
            for gap in default_replace_gaps(package_src, package_src):
                problems.append(f"{package_dir.relative_to(root)}: {gap}")

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
    problems.extend(description_string_to_framework(root))
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
