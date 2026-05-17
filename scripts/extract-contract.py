#!/usr/bin/env python3
"""Extract API contract from Coolify Laravel source code.

Reads models, controllers, and migrations to produce a machine-readable
contract JSON. This is the single source of truth for the Terraform
provider's client structs, schema attributes, validators, and defaults.

Usage:
    python3 scripts/extract-contract.py /path/to/coolify [--version v4.0.1]
"""

import argparse
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path


def extract_fillable(content: str) -> list[str]:
    """Extract $fillable array from a Laravel model."""
    m = re.search(r"protected\s+\$fillable\s*=\s*\[(.*?)\];", content, re.DOTALL)
    if not m:
        return []
    return re.findall(r"'([^']+)'", m.group(1))


def extract_guarded(content: str) -> list[str]:
    """Extract $guarded array from a Laravel model."""
    m = re.search(r"protected\s+\$guarded\s*=\s*\[(.*?)\];", content, re.DOTALL)
    if not m:
        return []
    return re.findall(r"'([^']+)'", m.group(1))


def extract_casts(content: str) -> dict[str, str]:
    """Extract $casts array or casts() method from a Laravel model."""
    casts = {}
    # Method style: protected function casts(): array { return [...]; }
    m = re.search(
        r"protected\s+function\s+casts\(\).*?return\s*\[(.*?)\];",
        content,
        re.DOTALL,
    )
    if m:
        for key, val in re.findall(r"'([^']+)'\s*=>\s*'([^']+)'", m.group(1)):
            casts[key] = val
        return casts
    # Property style: protected $casts = [...];
    m = re.search(r"protected\s+\$casts\s*=\s*\[(.*?)\];", content, re.DOTALL)
    if m:
        for key, val in re.findall(r"'([^']+)'\s*=>\s*'([^']+)'", m.group(1)):
            casts[key] = val
    return casts


def extract_hidden(content: str) -> list[str]:
    """Extract $hidden array from a Laravel model."""
    m = re.search(r"protected\s+\$hidden\s*=\s*\[(.*?)\];", content, re.DOTALL)
    if not m:
        return []
    return re.findall(r"'([^']+)'", m.group(1))


def extract_appends(content: str) -> list[str]:
    """Extract $appends array from a Laravel model."""
    m = re.search(r"protected\s+\$appends\s*=\s*\[(.*?)\];", content, re.DOTALL)
    if not m:
        return []
    return re.findall(r"'([^']+)'", m.group(1))


def extract_model_attributes(content: str) -> dict:
    """Extract $attributes array (model defaults) from a Laravel model."""
    attrs = {}
    m = re.search(
        r"protected\s+\$attributes\s*=\s*\[(.*?)\];", content, re.DOTALL
    )
    if not m:
        return attrs
    block = m.group(1)
    for key, val in re.findall(r"'([^']+)'\s*=>\s*(.+?)(?:,\s*$|\s*$)", block, re.MULTILINE):
        attrs[key] = _parse_php_value(val.strip().rstrip(","))
    return attrs


def _parse_php_value(val: str):
    """Parse a PHP literal value to a Python value."""
    val = val.strip()
    if val in ("true", "True"):
        return True
    if val in ("false", "False"):
        return False
    if val in ("null", "NULL"):
        return None
    if val.startswith("'") and val.endswith("'"):
        return val[1:-1]
    if val.startswith('"') and val.endswith('"'):
        return val[1:-1]
    try:
        return int(val)
    except ValueError:
        pass
    try:
        return float(val)
    except ValueError:
        pass
    return val


def _extract_up_method(content: str) -> str:
    """Extract only the up() method body from a migration.

    Laravel migrations have up() and down() methods. The down() method
    reverses the change and must not be processed, or its values overwrite
    the up() values (e.g., default(null) in up, default('0') in down).
    """
    m = re.search(
        r"public\s+function\s+up\s*\(\s*\)\s*(?::\s*void\s*)?\{(.*)",
        content,
        re.DOTALL,
    )
    if not m:
        return content  # fallback: use entire file
    body = m.group(1)
    depth = 1
    for i, ch in enumerate(body):
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return body[:i]
    return body


def parse_migration_columns(migration_dir: Path, table_name: str) -> dict:
    """Parse all migrations for a table and build the final column schema."""
    columns = {}
    migration_files = sorted(migration_dir.glob("*.php"))

    for mig_file in migration_files:
        content = mig_file.read_text(errors="replace")
        if table_name not in content:
            continue
        # Skip if it's about a different table with a similar name
        # e.g. 'application_settings' when looking for 'applications'
        if table_name == "applications" and "application_settings" in mig_file.name:
            continue

        up_content = _extract_up_method(content)
        _parse_create_table(up_content, table_name, columns)
        _parse_alter_table(up_content, table_name, columns)

    return columns


def _parse_create_table(content: str, table_name: str, columns: dict):
    """Parse Schema::create() for a table."""
    pattern = (
        r"Schema::create\(\s*'" + re.escape(table_name) + r"'"
        r".*?function\s*\(.*?\)\s*\{(.*?)\}\s*\)"
    )
    m = re.search(pattern, content, re.DOTALL)
    if not m:
        return
    _extract_column_defs(m.group(1), columns)


def _parse_alter_table(content: str, table_name: str, columns: dict):
    """Parse Schema::table() for ALTER operations."""
    pattern = (
        r"Schema::table\(\s*'" + re.escape(table_name) + r"'"
        r".*?function\s*\(.*?\)\s*\{(.*?)\}\s*\)"
    )
    for m in re.finditer(pattern, content, re.DOTALL):
        block = m.group(1)
        _extract_column_defs(block, columns)
        # Handle dropColumn
        for drop_match in re.finditer(
            r"\$table->dropColumn\(\s*'([^']+)'\s*\)", block
        ):
            columns.pop(drop_match.group(1), None)
        # Handle dropColumn with array
        for drop_match in re.finditer(
            r"\$table->dropColumn\(\s*\[(.*?)\]\s*\)", block, re.DOTALL
        ):
            for col in re.findall(r"'([^']+)'", drop_match.group(1)):
                columns.pop(col, None)


def _extract_column_defs(block: str, columns: dict):
    """Extract column definitions from a migration block."""
    # Match: $table->type('name')...;
    col_pattern = re.compile(
        r"\$table->(\w+)\(\s*'([^']+)'(?:\s*,\s*(?:(\d+)|(\[[^\]]*\])))?\s*\)(.*?);",
        re.DOTALL,
    )
    for match in col_pattern.finditer(block):
        col_type = match.group(1)
        col_name = match.group(2)
        col_length = match.group(3)  # numeric arg (e.g., string length)
        col_array = match.group(4)   # array arg (e.g., enum values)
        chain = match.group(5)

        # Skip non-column methods
        if col_type in (
            "dropColumn", "renameColumn", "unique", "index",
            "foreign", "dropForeign", "dropUnique", "dropIndex",
            "primary", "dropPrimary",
        ):
            continue

        type_map = {
            "string": "string",
            "text": "text",
            "longText": "longText",
            "boolean": "boolean",
            "integer": "integer",
            "bigInteger": "bigInteger",
            "unsignedBigInteger": "bigInteger",
            "timestamp": "timestamp",
            "enum": "enum",
            "json": "json",
            "float": "float",
            "decimal": "decimal",
            "smallInteger": "smallInteger",
            "tinyInteger": "tinyInteger",
            "foreignId": "bigInteger",
            "id": "bigInteger",
        }

        mapped_type = type_map.get(col_type, col_type)
        if mapped_type == col_type and col_type not in type_map:
            # Skip morphs, timestamps, softDeletes handled separately
            if col_type in ("morphs", "nullableMorphs", "timestamps", "softDeletes"):
                _handle_special_column(col_type, col_name, columns)
                continue
            continue

        nullable = "->nullable()" in chain or "->nullable(true)" in chain
        default = None
        default_match = re.search(r"->default\(\s*(.+?)\s*\)", chain)
        if default_match:
            default = _parse_php_value(default_match.group(1))

        col_info = {
            "type": mapped_type,
            "nullable": nullable,
            "default": default,
        }
        if col_length:
            col_info["max_length"] = int(col_length)
        elif mapped_type == "string":
            col_info["max_length"] = 255

        # Detect enum values from the captured array argument
        if col_type == "enum" and col_array:
            col_info["enum_values"] = re.findall(r"'([^']+)'", col_array)

        # If this is a ->change() migration and the column already exists,
        # merge only the explicitly-set properties instead of replacing
        # the entire entry. This preserves max_length, enum_values, etc.
        # from the original Schema::create() definition.
        is_change = "->change()" in chain
        if is_change and col_name in columns:
            existing = columns[col_name]
            existing["type"] = mapped_type
            if default_match:
                existing["default"] = default
            if "->nullable()" in chain or "->nullable(true)" in chain:
                existing["nullable"] = nullable
            if col_length:
                existing["max_length"] = int(col_length)
        else:
            columns[col_name] = col_info

    # Handle timestamps()
    if "$table->timestamps()" in block:
        for ts in ("created_at", "updated_at"):
            columns[ts] = {"type": "timestamp", "nullable": True, "default": None}

    # Handle softDeletes()
    if "$table->softDeletes()" in block:
        columns["deleted_at"] = {"type": "timestamp", "nullable": True, "default": None}


def _handle_special_column(col_type: str, col_name: str, columns: dict):
    """Handle morphs, timestamps, softDeletes."""
    if col_type in ("morphs", "nullableMorphs"):
        nullable = col_type == "nullableMorphs"
        columns[f"{col_name}_type"] = {
            "type": "string", "nullable": nullable, "default": None
        }
        columns[f"{col_name}_id"] = {
            "type": "bigInteger", "nullable": nullable, "default": None
        }


def extract_validation_rules(content: str) -> dict[str, dict]:
    """Extract validation rules from a controller file.

    Returns: {method_name: {field: rule_string}}
    """
    rules = {}
    # Match $request->validate([...]) blocks
    validate_pattern = re.compile(
        r"(?:function\s+(\w+)\s*\(.*?\).*?)?"
        r"\$request->validate\(\s*\[(.*?)\]\s*\)",
        re.DOTALL,
    )
    for match in validate_pattern.finditer(content):
        method = match.group(1) or "unknown"
        block = match.group(2)
        field_rules = {}
        for field, rule_array, rule_single, rule_double in re.findall(
            r"'([^']+)'\s*=>\s*(?:\[([^\]]*)\]|'([^']*)'|\"([^\"]*)\")",
            block,
        ):
            rule = rule_array or rule_single or rule_double
            field_rules[field] = _clean_rule(rule)
        if field_rules:
            rules[method] = field_rules
    return rules


def _clean_rule(rule: str) -> str:
    """Clean a Laravel validation rule string."""
    # Remove PHP expressions, keep string literals
    parts = []
    for part in re.findall(r"'([^']*)'", rule):
        parts.append(part)
    return "|".join(parts) if parts else rule.strip()


def extract_allowed_fields(content: str) -> dict[str, list[str]]:
    """Extract $allowedFields arrays from controller methods."""
    result = {}
    pattern = re.compile(
        r"\$allowedFields\s*=\s*\[(.*?)\];", re.DOTALL
    )
    # Find the enclosing method for context
    for match in pattern.finditer(content):
        fields = re.findall(r"'([^']+)'", match.group(1))
        # Try to find the method name above this position
        preceding = content[:match.start()]
        method_matches = re.findall(r"function\s+(\w+)\s*\(", preceding)
        method = method_matches[-1] if method_matches else "unknown"
        result[method] = fields
    return result


def extract_shared_validation(helpers_dir: Path) -> dict[str, str]:
    """Extract shared validation rules from bootstrap/helpers/api.php."""
    api_file = helpers_dir / "api.php"
    if not api_file.exists():
        return {}
    content = api_file.read_text(errors="replace")
    rules = {}
    # Look for sharedDataApplications, sharedDataDatabases, etc.
    pattern = re.compile(
        r"function\s+(\w+)\(\).*?return\s*\[(.*?)\];", re.DOTALL
    )
    for match in pattern.finditer(content):
        func_name = match.group(1)
        block = match.group(2)
        for field, rule_parts in re.findall(
            r"'([^']+)'\s*=>\s*\[([^\]]*)\]", block
        ):
            rule_str = _clean_rule(rule_parts)
            if rule_str:
                rules[field] = rule_str
    return rules


def extract_enums(app_dir: Path) -> dict[str, list[str]]:
    """Extract enum classes from app/Enums/."""
    enums_dir = app_dir / "Enums"
    result = {}
    if not enums_dir.exists():
        return result
    for enum_file in enums_dir.glob("*.php"):
        content = enum_file.read_text(errors="replace")
        # Get enum name
        name_match = re.search(r"enum\s+(\w+)", content)
        if not name_match:
            continue
        name = name_match.group(1)
        # Get case values
        cases = re.findall(r"case\s+\w+\s*=\s*'([^']+)'", content)
        if cases:
            result[name] = cases
    return result


def extract_validation_patterns(app_dir: Path) -> dict[str, str]:
    """Extract validation patterns from app/Support/ValidationPatterns.php."""
    vp_file = app_dir / "Support" / "ValidationPatterns.php"
    if not vp_file.exists():
        return {}
    content = vp_file.read_text(errors="replace")
    patterns = {}
    # Match: const PATTERN_NAME = '/regex/';
    for name, regex in re.findall(
        r"const\s+(\w+_PATTERN)\s*=\s*['\"](.+?)['\"]", content
    ):
        patterns[name] = regex
    # Match: regex in method returns
    for name, regex in re.findall(
        r"'regex:(/[^']+/)'", content
    ):
        patterns[f"inline_{len(patterns)}"] = regex
    return patterns


def build_model_contract(
    model_path: Path,
    migration_dir: Path,
    table_name: str,
    casts_override: dict | None = None,
) -> dict:
    """Build a complete model contract from model file + migrations."""
    content = model_path.read_text(errors="replace")

    fillable = extract_fillable(content)
    guarded = extract_guarded(content)
    casts = casts_override or extract_casts(content)
    hidden = extract_hidden(content)
    appends = extract_appends(content)
    model_defaults = extract_model_attributes(content)

    # Get column schema from migrations
    columns = parse_migration_columns(migration_dir, table_name)

    # Build field map: merge migration schema with model metadata
    fields = {}
    all_field_names = set(fillable) | set(columns.keys())
    # Exclude internal-only fields from the contract
    internal_fields = {
        "id", "created_at", "updated_at", "deleted_at",
        "team_id",
    }

    for fname in sorted(all_field_names - internal_fields):
        col = columns.get(fname, {})
        is_nullable = col.get("nullable", False)
        default_val = model_defaults.get(fname, col.get("default"))
        is_sensitive = casts.get(fname) == "encrypted"
        is_hidden = fname in hidden
        field_info = {
            "type": col.get("type", "string"),
            "nullable": is_nullable,
            "default": default_val,
            "cast": casts.get(fname),
            "sensitive": is_sensitive,
            "fillable": fname in fillable,
            # Flatten helper recommendation based on DB schema:
            # - "unconditional": NOT NULL with default, API always returns value
            # - "or_clear": NULLABLE without default, must detect external clearing
            # - "if_configured": sensitive/hidden field, empty is ambiguous
            "flatten_hint": (
                "if_configured" if (is_sensitive or is_hidden) else
                "or_clear" if is_nullable else
                "unconditional"
            ),
        }
        if "max_length" in col:
            field_info["max_length"] = col["max_length"]
        if "enum_values" in col:
            field_info["enum_values"] = col["enum_values"]
        fields[fname] = field_info

    return {
        "table": table_name,
        "fillable": fillable,
        "guarded": guarded,
        "hidden": hidden,
        "appends": appends,
        "fields": fields,
    }


def extract_contract(coolify_dir: str, version: str = "unknown") -> dict:
    """Extract the full API contract from a Coolify source directory."""
    root = Path(coolify_dir)
    models_dir = root / "app" / "Models"
    controllers_dir = root / "app" / "Http" / "Controllers" / "Api"
    migration_dir = root / "database" / "migrations"
    app_dir = root / "app"
    helpers_dir = root / "bootstrap" / "helpers"

    if not models_dir.exists():
        print(f"Error: {models_dir} not found", file=sys.stderr)
        sys.exit(1)

    contract = {
        "version": version,
        "extracted_from": f"coollabsio/coolify@{version}",
        "extracted_at": datetime.now(timezone.utc).isoformat(),
        "models": {},
        "endpoints": {},
        "enums": {},
        "validation_patterns": {},
        "shared_validation_rules": {},
    }

    # Model definitions
    model_table_map = {
        "Application": "applications",
        "Server": "servers",
        "ServerSetting": "server_settings",
        "Project": "projects",
        "Environment": "environments",
        "Service": "services",
        "StandalonePostgresql": "standalone_postgresqls",
        "StandaloneMysql": "standalone_mysqls",
        "StandaloneMariadb": "standalone_mariadbs",
        "StandaloneMongodb": "standalone_mongodbs",
        "StandaloneRedis": "standalone_redis",
        "StandaloneClickhouse": "standalone_clickhouses",
        "StandaloneKeydb": "standalone_keydbs",
        "StandaloneDragonfly": "standalone_dragonflies",
        "EnvironmentVariable": "environment_variables",
        "PrivateKey": "private_keys",
        "LocalPersistentVolume": "local_persistent_volumes",
        "ScheduledTask": "scheduled_tasks",
        "GithubApp": "github_apps",
        "S3Storage": "s3_storages",
        "CloudProviderToken": "cloud_provider_tokens",
        "ScheduledDatabaseBackup": "scheduled_database_backups",
    }

    for model_name, table_name in model_table_map.items():
        model_file = models_dir / f"{model_name}.php"
        if model_file.exists():
            contract["models"][model_name] = build_model_contract(
                model_file, migration_dir, table_name
            )

    # Extract ApplicationSetting separately and attach to Application
    app_setting_file = models_dir / "ApplicationSetting.php"
    if app_setting_file.exists():
        settings = build_model_contract(
            app_setting_file, migration_dir, "application_settings"
        )
        if "Application" in contract["models"]:
            contract["models"]["Application"]["settings_fields"] = settings["fields"]

    # Controller endpoints
    for ctrl_file in sorted(controllers_dir.glob("*.php")):
        if ctrl_file.name == "OpenApi.php":
            continue
        content = ctrl_file.read_text(errors="replace")
        allowed = extract_allowed_fields(content)
        for method, fields in allowed.items():
            endpoint_key = f"{ctrl_file.stem}::{method}"
            contract["endpoints"][endpoint_key] = {
                "allowed_fields": fields,
            }

    # Enums
    contract["enums"] = extract_enums(app_dir)

    # Validation patterns
    contract["validation_patterns"] = extract_validation_patterns(app_dir)

    # Shared validation rules
    contract["shared_validation_rules"] = extract_shared_validation(helpers_dir)

    return contract


def main():
    parser = argparse.ArgumentParser(
        description="Extract API contract from Coolify source code"
    )
    parser.add_argument(
        "coolify_dir",
        help="Path to Coolify source directory",
    )
    parser.add_argument(
        "--version",
        default="unknown",
        help="Coolify version tag (e.g., v4.0.1)",
    )
    parser.add_argument(
        "--output",
        default=None,
        help="Output file path (default: stdout)",
    )
    args = parser.parse_args()

    contract = extract_contract(args.coolify_dir, args.version)
    output = json.dumps(contract, indent=2, sort_keys=False)

    if args.output:
        Path(args.output).write_text(output + "\n")
        print(f"Contract written to {args.output}", file=sys.stderr)
    else:
        print(output)


if __name__ == "__main__":
    main()
