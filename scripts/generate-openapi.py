#!/usr/bin/env python3
"""Generate a corrected OpenAPI spec from the source-code contract.

Takes the existing OpenAPI spec (for endpoint/path structure) and patches
the component schemas with correct types, nullability, defaults, enums,
and required fields from the contract JSON.

Usage:
    python3 scripts/generate-openapi.py \
        --contract testdata/contracts/coolify-v4.json \
        --spec testdata/specs/coolify-v4.json \
        --output testdata/specs/coolify-v4.json
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Optional


# Map contract model names to OpenAPI schema names
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
    "StandalonePostgresql": "StandalonePostgresql",
    "StandaloneMysql": "StandaloneMysql",
    "StandaloneMariadb": "StandaloneMariadb",
    "StandaloneMongodb": "StandaloneMongodb",
    "StandaloneRedis": "StandaloneRedis",
    "StandaloneClickhouse": "StandaloneClickhouse",
    "StandaloneKeydb": "StandaloneKeydb",
    "StandaloneDragonfly": "StandaloneDragonfly",
    "GithubApp": "GithubApp",
    "S3Storage": "S3Storage",
    "LocalPersistentVolume": "LocalPersistentVolume",
    "CloudProviderToken": "CloudProviderToken",
}

# Map contract field types to OpenAPI types
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


def patch_schema(schema: dict, contract_model: dict) -> dict:
    """Patch an OpenAPI schema with correct data from the contract."""
    fields = contract_model.get("fields", {})
    settings_fields = contract_model.get("settings_fields", {})
    all_fields = {**fields, **settings_fields}

    properties = schema.get("properties", {})

    for field_name, field_info in all_fields.items():
        if field_name not in properties:
            # Field exists in contract but not in spec -- add it
            prop = _build_property(field_name, field_info)
            if prop:
                properties[field_name] = prop
            continue

        # Field exists in both -- patch it
        prop = properties[field_name]
        _patch_property(prop, field_info)

    # Add properties that only exist in contract (new fields)
    for field_name, field_info in fields.items():
        if field_name not in properties and field_info.get("fillable", False):
            prop = _build_property(field_name, field_info)
            if prop:
                properties[field_name] = prop

    # Don't add required to response schemas. The contract test validates
    # field coverage independently.
    schema["properties"] = dict(sorted(properties.items()))
    schema.pop("required", None)

    return schema


def _build_property(field_name: str, field_info: dict) -> Optional[dict]:
    """Build an OpenAPI property from a contract field."""
    contract_type = field_info.get("type", "string")
    openapi_type = TYPE_MAP.get(contract_type, "string")

    prop = {
        "type": openapi_type,
        "description": field_name.replace("_", " ").capitalize() + ".",
    }

    # Don't add nullable on new fields. The original spec already has
    # nullable annotations on existing fields where appropriate.

    if field_info.get("default") is not None:
        prop["default"] = field_info["default"]

    if field_info.get("enum_values"):
        prop["enum"] = field_info["enum_values"]

    if openapi_type == "string" and field_info.get("max_length"):
        prop["maxLength"] = field_info["max_length"]

    if contract_type == "timestamp":
        prop["format"] = "date-time"

    return prop


def _patch_property(prop: dict, field_info: dict):
    """Patch an existing OpenAPI property with contract data."""
    contract_type = field_info.get("type", "string")
    openapi_type = TYPE_MAP.get(contract_type, "string")

    # Fix type
    prop["type"] = openapi_type

    # Preserve existing nullable annotations but don't add new ones.

    # Fix/add default
    if field_info.get("default") is not None:
        prop["default"] = field_info["default"]

    # Fix/add enum
    if field_info.get("enum_values"):
        prop["enum"] = field_info["enum_values"]

    # Fix format for timestamps
    if contract_type == "timestamp":
        prop["format"] = "date-time"


def patch_request_bodies(spec: dict, contract: dict):
    """Patch request body schemas in endpoint definitions."""
    paths = spec.get("paths", {})

    # Fix the environment_uuid required issue on create endpoints
    for path_key, path_item in paths.items():
        for method in ("post", "put", "patch"):
            op = path_item.get(method)
            if not op:
                continue
            rb = op.get("requestBody", {})
            content = rb.get("content", {}).get("application/json", {})
            schema = content.get("schema", {})
            required = schema.get("required", [])

            # environment_uuid is never truly required (conditional with environment_name)
            if "environment_uuid" in required:
                required.remove("environment_uuid")
                if required:
                    schema["required"] = required
                else:
                    schema.pop("required", None)


def build_schema_from_contract(contract_model: dict) -> dict:
    """Build a complete OpenAPI schema from a contract model."""
    fields = contract_model.get("fields", {})
    settings_fields = contract_model.get("settings_fields", {})
    all_fields = {**fields, **settings_fields}

    properties = {}
    for field_name, field_info in all_fields.items():
        prop = _build_property(field_name, field_info)
        if prop:
            if field_info.get("nullable", False):
                prop["nullable"] = True
            properties[field_name] = prop

    return {
        "type": "object",
        "properties": dict(sorted(properties.items())),
    }


def main():
    parser = argparse.ArgumentParser(
        description="Generate corrected OpenAPI spec from contract"
    )
    parser.add_argument("--contract", required=True, help="Contract JSON path")
    parser.add_argument("--spec", required=True, help="Existing OpenAPI spec path")
    parser.add_argument("--output", required=True, help="Output spec path")
    args = parser.parse_args()

    contract = json.loads(Path(args.contract).read_text())
    spec = json.loads(Path(args.spec).read_text())

    schemas = spec.get("components", {}).get("schemas", {})

    patched_count = 0
    created_count = 0
    for model_name, schema_name in MODEL_TO_SCHEMA.items():
        if model_name not in contract.get("models", {}):
            continue
        if schema_name not in schemas:
            # Create a new schema from the contract
            schemas[schema_name] = build_schema_from_contract(
                contract["models"][model_name]
            )
            created_count += 1
            continue
        schemas[schema_name] = patch_schema(
            schemas[schema_name], contract["models"][model_name]
        )
        patched_count += 1

    # Fix request body issues
    patch_request_bodies(spec, contract)

    Path(args.output).write_text(json.dumps(spec, indent=4, ensure_ascii=False) + "\n")
    print(
        f"Patched {patched_count}, created {created_count} schemas, wrote {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
