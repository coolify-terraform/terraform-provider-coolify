#!/usr/bin/env python3
"""Generate the API contract accuracy guide.

Reads the source-derived contract JSON and pinned OpenAPI spec, then
produces the Markdown guide template used for the API contract accuracy
documentation. Reusable public schemas are compared field-by-field, and
contract-only or inline-only models are documented separately.

Usage:
    python3 scripts/generate-contract-matrix.py
"""

import json
import re
import sys
from pathlib import Path
from typing import Optional

ROOT = Path(__file__).resolve().parent.parent
CONTRACT_PATH = ROOT / "testdata" / "contracts" / "coolify-v4.json"
SPEC_PATH = ROOT / "testdata" / "specs" / "coolify-v4.json"
OUTPUT_PATH = ROOT / "templates" / "guides" / "api-contract-accuracy.md.tmpl"
CLIENT_DIR = ROOT / "internal" / "client"

# Preferred documentation order, with the reusable OpenAPI schema name to compare when present.
MODEL_TO_SCHEMA = {
    "Application": "Application",
    "Server": "Server",
    "ServerSetting": "ServerSetting",
    "Project": "Project",
    "Environment": "Environment",
    "Service": "Service",
    "EnvironmentVariable": "EnvironmentVariable",
    "PrivateKey": "PrivateKey",
    "ScheduledTask": "ScheduledTask",
    "ScheduledDatabaseBackup": "ScheduledDatabaseBackup",
}

# Contract models that intentionally have no client JSON mapping surface.
CLIENT_JSON_TAG_OVERRIDES = {
    "S3Storage": set(),
}

# Models we keep in the contract-only / inline-only section even though the
# pinned route spec may mention them elsewhere.
CONTRACT_ONLY_MODELS = {
    "CloudProviderToken",
    "GithubApp",
    "S3Storage",
}

# Map contract field types to a normalised display type.
TYPE_MAP = {
    "string": "string",
    "text": "string",
    "longText": "string",
    "boolean": "boolean",
    "integer": "integer",
    "bigInteger": "integer",
    "smallInteger": "integer",
    "tinyInteger": "integer",
    "float": "number",
    "decimal": "number",
    "timestamp": "string",
    "json": "object",
    "enum": "string",
}

# Map OpenAPI type keywords to the same normalised types.
OPENAPI_TYPE_MAP = {
    "string": "string",
    "boolean": "boolean",
    "integer": "integer",
    "number": "number",
    "object": "object",
    "array": "array",
}


def _extract_json_tags(go_file: Path) -> set[str]:
    """Extract all json struct-tag names from a Go source file."""
    tags: set[str] = set()
    text = go_file.read_text()
    for m in re.finditer(r'`json:"([^"]+)"`', text):
        raw = m.group(1)
        name = raw.split(",")[0]
        if name and name != "-":
            tags.add(name)
    return tags


def _load_client_json_tags() -> set[str]:
    """Load all JSON tags from all client Go files."""
    tags: set[str] = set()
    for f in CLIENT_DIR.glob("*.go"):
        tags |= _extract_json_tags(f)
    return tags


def _normalise_spec_type(prop: dict) -> str:
    """Return a normalised type string from an OpenAPI property."""
    t = prop.get("type", "")
    if isinstance(t, list):
        # OpenAPI 3.1 type arrays like ["string", "null"]
        types = [x for x in t if x != "null"]
        t = types[0] if types else "string"
    return OPENAPI_TYPE_MAP.get(t, t)


def _nullable_from_spec(prop: dict) -> bool:
    """Check if a spec property is nullable."""
    if prop.get("nullable"):
        return True
    t = prop.get("type", "")
    if isinstance(t, list) and "null" in t:
        return True
    return False


def _format_default(val) -> str:
    if val is None:
        return "-"
    if isinstance(val, bool):
        return str(val).lower()
    s = str(val)
    # Escape Go template syntax so tfplugindocs doesn't try to evaluate it
    if "{{" in s:
        s = s.replace("{{", "{ {").replace("}}", "} }")
    return s


REVIEWED_NULLABILITY_DRIFT = {
    "Application": {
        "build_command",
        "custom_labels",
        "custom_network_aliases",
        "custom_nginx_configuration",
        "description",
        "docker_compose_custom_build_command",
        "docker_compose_custom_start_command",
        "docker_compose_domains",
        "docker_compose_raw",
        "dockerfile",
        "dockerfile_target_build",
        "health_check_command",
        "health_check_host",
        "http_basic_auth_password",
        "http_basic_auth_username",
        "install_command",
        "manual_webhook_secret_bitbucket",
        "manual_webhook_secret_gitea",
        "manual_webhook_secret_github",
        "manual_webhook_secret_gitlab",
        "post_deployment_command_container",
        "pre_deployment_command_container",
        "publish_directory",
        "redirect",
        "start_command",
        "watch_paths",
    },
    "EnvironmentVariable": {"value"},
    "PrivateKey": {"description"},
    "Project": {"description"},
    "Server": {"description"},
}


def _compare_field(
    model_name: str,
    field_name: str,
    contract_field: dict,
    spec_props: dict,
    client_json_tags: set[str],
) -> dict:
    """Compare a single field between contract and spec."""
    contract_type = TYPE_MAP.get(contract_field.get("type", "string"), "string")
    contract_nullable = contract_field.get("nullable", False)
    contract_default = contract_field.get("default")

    spec_prop = spec_props.get(field_name)
    if spec_prop is None:
        return {
            "field": field_name,
            "contract_type": contract_type,
            "spec_type": "-",
            "nullable_match": "-",
            "default": _format_default(contract_default),
            "client_json_mapping": "mapped" if field_name in client_json_tags else "n/a",
        }

    spec_type = _normalise_spec_type(spec_prop)
    spec_nullable = _nullable_from_spec(spec_prop)

    type_match = contract_type == spec_type
    nullable_match = contract_nullable == spec_nullable
    reviewed_drift = (
        not nullable_match
        and field_name in REVIEWED_NULLABILITY_DRIFT.get(model_name, set())
    )

    return {
        "field": field_name,
        "contract_type": contract_type,
        "spec_type": spec_type,
        "type_match": type_match,
        "nullable_match": "yes" if nullable_match else ("reviewed drift" if reviewed_drift else "**WRONG**"),
        "default": _format_default(contract_default),
        "client_json_mapping": "mapped" if field_name in client_json_tags else "n/a",
    }


def _model_intro(model_name: str, spec_schema: Optional[dict]) -> list[str]:
    if model_name == "ScheduledDatabaseBackup":
        return [
            "This section compares the internal source-derived backup model against the public backup request bodies in the pinned spec.",
            "Coolify stores the relation as `s3_storage_id` internally, while the public API accepts `s3_storage_uuid` on request bodies.",
            "That identifier translation is expected and does not imply a missing top-level S3 CRUD API.",
            "",
        ]
    if spec_schema is None and model_name in CONTRACT_ONLY_MODELS:
        return [
            "This model exists in the extracted source contract but not as a reusable public OpenAPI schema.",
            "Treat it as implementation detail coverage, not proof of a standalone public API surface.",
            "",
        ]
    return []


def _build_model_table(
    model_name: str,
    contract_model: dict,
    spec_schema: Optional[dict],
    client_json_tags: set[str],
) -> list[str]:
    """Build Markdown table lines for one model."""
    fields = contract_model.get("fields", {})
    settings_fields = contract_model.get("settings_fields", {})
    all_fields = {**fields, **settings_fields}

    spec_props = {}
    if spec_schema:
        spec_props = spec_schema.get("properties", {})

    client_json_tags = CLIENT_JSON_TAG_OVERRIDES.get(model_name, client_json_tags)

    rows: list[dict] = []
    for field_name in sorted(all_fields):
        rows.append(
            _compare_field(model_name, field_name, all_fields[field_name], spec_props, client_json_tags)
        )

    # Also report spec-only fields (in spec but not in contract).
    for field_name in sorted(spec_props):
        if field_name not in all_fields:
            spec_type = _normalise_spec_type(spec_props[field_name])
            rows.append(
                {
                    "field": field_name,
                    "contract_type": "-",
                    "spec_type": spec_type,
                    "nullable_match": "-",
                    "default": "-",
                    "client_json_mapping": "mapped" if field_name in client_json_tags else "n/a",
                }
            )

    # Compute stats.
    total = len(rows)
    type_ok = sum(1 for r in rows if r.get("type_match", True))
    nullable_ok = sum(1 for r in rows if r.get("nullable_match") in ("yes", "reviewed drift", "-"))
    client_mapping_ok = sum(1 for r in rows if r["client_json_mapping"] == "mapped")

    lines: list[str] = []
    lines.append(f"## {model_name}")
    lines.append("")
    lines.extend(_model_intro(model_name, spec_schema))
    lines.append(
        f"Fields: {total} | Type matches: {type_ok}/{total} "
        f"| Nullable matches: {nullable_ok}/{total} "
        f"| Client JSON mappings: {client_mapping_ok}/{total}"
    )
    lines.append("")
    lines.append(
        "| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |"
    )
    lines.append(
        "|-------|:---:|:---:|:---:|:---:|---------|:---:|"
    )
    for r in rows:
        ct = r["contract_type"]
        st = r["spec_type"]
        if r.get("type_match") is None:
            tm = "-"
        elif r["type_match"]:
            tm = "yes"
        else:
            tm = "**WRONG**"
        nm = r["nullable_match"]
        d = r["default"]
        p = r["client_json_mapping"]
        lines.append(f"| {r['field']} | {ct} | {st} | {tm} | {nm} | {d} | {p} |")

    lines.append("")
    return lines


def main():
    if not CONTRACT_PATH.exists():
        print(f"Contract not found: {CONTRACT_PATH}", file=sys.stderr)
        sys.exit(1)
    if not SPEC_PATH.exists():
        print(f"Spec not found: {SPEC_PATH}", file=sys.stderr)
        sys.exit(1)

    contract = json.loads(CONTRACT_PATH.read_text())
    spec = json.loads(SPEC_PATH.read_text())
    schemas = spec.get("components", {}).get("schemas", {})
    client_json_tags = _load_client_json_tags()

    out: list[str] = []
    out.append("---")
    out.append('page_title: "API Contract Accuracy"')
    out.append('subcategory: "Guides"')
    out.append(
        'description: "Comparison of Coolify OpenAPI spec vs real source code contract."'
    )
    out.append("---")
    out.append("")
    out.append("# API Contract Accuracy")
    out.append("")
    out.append(
        "This page compares the pinned reusable OpenAPI schemas with the source-derived"
    )
    out.append("Coolify contract extracted from the real application code.")
    out.append("")
    out.append("> The source-derived contract is the field-level source of truth. The pinned OpenAPI spec is useful for reusable public schemas and route inventory, but some contract models only exist as internal implementation details or inline request bodies.")
    out.append("> `reviewed drift` means the pinned spec and source contract disagree on nullability, but the provider already handles the field safely and no runtime fix is needed.")
    out.append("> `mapped` means the field name appears in the provider's internal client JSON structs. It does not guarantee Terraform schema exposure, read-after-write round trips, or full CRUD behavior.")
    out.append("")
    out.append(
        f"Contract version: `{contract.get('version', 'unknown')}` | "
        f"Extracted from: `{contract.get('extracted_from', 'unknown')}`"
    )
    out.append("")

    def accumulate(section: list[str], stats: dict[str, int]):
        for line in section:
            if not line.startswith("Fields:"):
                continue
            parts = line.split("|")
            for p in parts:
                p = p.strip()
                if p.startswith("Fields:"):
                    stats["total_fields"] += int(p.split(":")[1].strip())
                elif p.startswith("Type matches:"):
                    n, _ = p.split(":")[1].strip().split("/")
                    stats["total_type_ok"] += int(n)
                elif p.startswith("Nullable matches:"):
                    n, _ = p.split(":")[1].strip().split("/")
                    stats["total_nullable_ok"] += int(n)
                elif p.startswith("Client JSON mappings:"):
                    n, _ = p.split(":")[1].strip().split("/")
                    stats["total_client_mapping"] += int(n)

    public_stats = {
        "total_fields": 0,
        "total_type_ok": 0,
        "total_nullable_ok": 0,
        "total_client_mapping": 0,
    }
    contract_only_stats = {
        "total_fields": 0,
        "total_type_ok": 0,
        "total_nullable_ok": 0,
        "total_client_mapping": 0,
    }

    public_sections: list[list[str]] = []
    contract_only_sections: list[list[str]] = []

    for model_name, schema_name in sorted(MODEL_TO_SCHEMA.items()):
        contract_model = contract.get("models", {}).get(model_name)
        if contract_model is None:
            continue
        spec_schema = schemas.get(schema_name)
        section = _build_model_table(model_name, contract_model, spec_schema, client_json_tags)
        if spec_schema is None:
            contract_only_sections.append(section)
            accumulate(section, contract_only_stats)
        else:
            public_sections.append(section)
            accumulate(section, public_stats)

    for model_name in sorted(contract.get("models", {})):
        if model_name in MODEL_TO_SCHEMA:
            continue
        contract_model = contract["models"][model_name]
        section = _build_model_table(model_name, contract_model, None, client_json_tags)
        contract_only_sections.append(section)
        accumulate(section, contract_only_stats)

    out.append("## Summary")
    out.append("")
    out.append(f"| Metric | Count |")
    out.append(f"|--------|------:|")
    out.append(f"| Public schema fields compared | {public_stats['total_fields']} |")
    out.append(f"| Public schema type matches | {public_stats['total_type_ok']}/{public_stats['total_fields']} |")
    out.append(f"| Public schema nullable matches | {public_stats['total_nullable_ok']}/{public_stats['total_fields']} |")
    out.append(f"| Public schema client JSON mappings | {public_stats['total_client_mapping']}/{public_stats['total_fields']} |")
    out.append(f"| Reusable public schemas compared | {len(public_sections)} |")
    out.append(f"| Contract-only / inline-only models documented | {len(contract_only_sections)} |")
    out.append("")
    out.append("---")
    out.append("")
    out.append("## Reusable Public Schemas")
    out.append("")

    for section in public_sections:
        out.extend(section)

    if contract_only_sections:
        out.append("## Contract-Only or Inline-Only Models")
        out.append("")
        out.append("These sections document source-derived models that do not map cleanly to reusable public OpenAPI component schemas.")
        out.append("")
        for section in contract_only_sections:
            out.extend(section)

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text("\n".join(out) + "\n")
    total_fields = public_stats["total_fields"] + contract_only_stats["total_fields"]
    total_models = len(public_sections) + len(contract_only_sections)
    print(f"Generated {OUTPUT_PATH} ({total_fields} fields across {total_models} models)", file=sys.stderr)


if __name__ == "__main__":
    main()
