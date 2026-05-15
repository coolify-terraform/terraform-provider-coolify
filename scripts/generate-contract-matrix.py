#!/usr/bin/env python3
"""Generate a contract accuracy matrix comparing source-code contract vs OpenAPI spec.

Reads the extracted contract JSON and the (patched) OpenAPI spec, then
produces a Markdown guide template showing field-by-field accuracy for
each model that exists in both files.

Usage:
    python3 scripts/generate-contract-matrix.py
"""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONTRACT_PATH = ROOT / "testdata" / "contracts" / "coolify-v4.json"
SPEC_PATH = ROOT / "testdata" / "specs" / "coolify-v4.json"
OUTPUT_PATH = ROOT / "templates" / "guides" / "api-contract-accuracy.md.tmpl"
CLIENT_DIR = ROOT / "internal" / "client"

# Maps contract model names to OpenAPI schema names (only models in both).
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
}

# Contract models that intentionally have no public provider surface.
PROVIDER_TAG_OVERRIDES = {
    "S3Storage": set(),
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


def _load_provider_tags() -> set[str]:
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


def _compare_field(
    field_name: str,
    contract_field: dict,
    spec_props: dict,
    provider_tags: set[str],
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
            "provider": "supported" if field_name in provider_tags else "n/a",
        }

    spec_type = _normalise_spec_type(spec_prop)
    spec_nullable = _nullable_from_spec(spec_prop)

    type_match = contract_type == spec_type
    nullable_match = contract_nullable == spec_nullable

    return {
        "field": field_name,
        "contract_type": contract_type,
        "spec_type": spec_type,
        "type_match": type_match,
        "nullable_match": "yes" if nullable_match else "**WRONG**",
        "default": _format_default(contract_default),
        "provider": "supported" if field_name in provider_tags else "n/a",
    }


def _build_model_table(
    model_name: str,
    contract_model: dict,
    spec_schema: dict | None,
    provider_tags: set[str],
) -> list[str]:
    """Build Markdown table lines for one model."""
    fields = contract_model.get("fields", {})
    settings_fields = contract_model.get("settings_fields", {})
    all_fields = {**fields, **settings_fields}

    spec_props = {}
    if spec_schema:
        spec_props = spec_schema.get("properties", {})

    provider_tags = PROVIDER_TAG_OVERRIDES.get(model_name, provider_tags)

    rows: list[dict] = []
    for field_name in sorted(all_fields):
        rows.append(
            _compare_field(field_name, all_fields[field_name], spec_props, provider_tags)
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
                    "provider": "supported" if field_name in provider_tags else "n/a",
                }
            )

    # Compute stats.
    total = len(rows)
    type_ok = sum(1 for r in rows if r.get("type_match", True))
    nullable_ok = sum(1 for r in rows if r.get("nullable_match") in ("yes", "-"))
    provider_ok = sum(1 for r in rows if r["provider"] == "supported")

    lines: list[str] = []
    lines.append(f"## {model_name}")
    lines.append("")
    lines.append(
        f"Fields: {total} | Type matches: {type_ok}/{total} "
        f"| Nullable matches: {nullable_ok}/{total} "
        f"| Provider coverage: {provider_ok}/{total}"
    )
    lines.append("")
    lines.append(
        "| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Provider |"
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
        p = r["provider"]
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
    provider_tags = _load_provider_tags()

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
        "This page shows the accuracy of the Coolify OpenAPI specification compared"
    )
    out.append("to the real API behavior extracted from the Coolify source code.")
    out.append("")
    out.append(
        f"Contract version: `{contract.get('version', 'unknown')}` | "
        f"Extracted from: `{contract.get('extracted_from', 'unknown')}`"
    )
    out.append("")

    # Summary counts.
    total_fields = 0
    total_type_ok = 0
    total_nullable_ok = 0
    total_provider = 0

    model_sections: list[list[str]] = []
    for model_name, schema_name in sorted(MODEL_TO_SCHEMA.items()):
        contract_model = contract.get("models", {}).get(model_name)
        if contract_model is None:
            continue
        spec_schema = schemas.get(schema_name)
        section = _build_model_table(model_name, contract_model, spec_schema, provider_tags)
        model_sections.append(section)

        # Parse stats from the summary line.
        for line in section:
            if line.startswith("Fields:"):
                parts = line.split("|")
                for p in parts:
                    p = p.strip()
                    if p.startswith("Fields:"):
                        total_fields += int(p.split(":")[1].strip())
                    elif p.startswith("Type matches:"):
                        n, d = p.split(":")[1].strip().split("/")
                        total_type_ok += int(n)
                    elif p.startswith("Nullable matches:"):
                        n, d = p.split(":")[1].strip().split("/")
                        total_nullable_ok += int(n)
                    elif p.startswith("Provider coverage:"):
                        n, d = p.split(":")[1].strip().split("/")
                        total_provider += int(n)

    # Also generate sections for contract models NOT in the spec.
    for model_name in sorted(contract.get("models", {})):
        if model_name in MODEL_TO_SCHEMA:
            continue
        contract_model = contract["models"][model_name]
        section = _build_model_table(model_name, contract_model, None, provider_tags)
        model_sections.append(section)

        for line in section:
            if line.startswith("Fields:"):
                parts = line.split("|")
                for p in parts:
                    p = p.strip()
                    if p.startswith("Fields:"):
                        total_fields += int(p.split(":")[1].strip())
                    elif p.startswith("Type matches:"):
                        n, d = p.split(":")[1].strip().split("/")
                        total_type_ok += int(n)
                    elif p.startswith("Provider coverage:"):
                        n, d = p.split(":")[1].strip().split("/")
                        total_provider += int(n)

    out.append("## Summary")
    out.append("")
    out.append(f"| Metric | Count |")
    out.append(f"|--------|------:|")
    out.append(f"| Total fields | {total_fields} |")
    out.append(f"| Type matches | {total_type_ok}/{total_fields} |")
    out.append(f"| Nullable matches | {total_nullable_ok}/{total_fields} |")
    out.append(f"| Provider coverage | {total_provider}/{total_fields} |")
    out.append(f"| Models in contract | {len(contract.get('models', {}))} |")
    out.append(
        f"| Models in spec | {len(schemas)} |"
    )
    out.append("")
    out.append("---")
    out.append("")

    for section in model_sections:
        out.extend(section)

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text("\n".join(out) + "\n")
    print(f"Generated {OUTPUT_PATH} ({total_fields} fields across {len(model_sections)} models)", file=sys.stderr)


if __name__ == "__main__":
    main()
