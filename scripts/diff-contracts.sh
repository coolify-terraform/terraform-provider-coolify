#!/usr/bin/env bash
# Diff two Coolify contract JSON files.
#
# Usage:
#   scripts/diff-contracts.sh testdata/contracts/coolify-v4.0.1.json testdata/contracts/coolify-v4.0.2.json
#
# Shows:
#   - New/removed models
#   - New/removed fields per model
#   - Changed defaults, types, nullability
#   - New/removed enums

set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: scripts/diff-contracts.sh <old-contract.json> <new-contract.json>" >&2
    exit 1
fi

OLD="$1"
NEW="$2"

if ! command -v python3 >/dev/null 2>&1; then
    echo "ERROR: python3 is required to run scripts/diff-contracts.sh. Install Python 3.9+ and re-run." >&2
    exit 1
fi

if [[ ! -f "$OLD" ]]; then
    echo "ERROR: contract file not found: $OLD" >&2
    exit 1
fi

if [[ ! -f "$NEW" ]]; then
    echo "ERROR: contract file not found: $NEW" >&2
    exit 1
fi

python3 - "$OLD" "$NEW" <<'PYTHON'
import json
import sys

old = json.load(open(sys.argv[1]))
new = json.load(open(sys.argv[2]))

changes = 0

# Models
old_models = set(old.get("models", {}).keys())
new_models = set(new.get("models", {}).keys())

added_models = new_models - old_models
removed_models = old_models - new_models
if added_models:
    print(f"\n+ New models: {', '.join(sorted(added_models))}")
    changes += len(added_models)
if removed_models:
    print(f"\n- Removed models: {', '.join(sorted(removed_models))}")
    changes += len(removed_models)

# Fields per model
for model in sorted(old_models & new_models):
    old_fields = set(old["models"][model].get("fields", {}).keys())
    new_fields = set(new["models"][model].get("fields", {}).keys())
    added = new_fields - old_fields
    removed = old_fields - new_fields
    if added:
        print(f"\n+ {model}: new fields: {', '.join(sorted(added))}")
        changes += len(added)
    if removed:
        print(f"\n- {model}: removed fields: {', '.join(sorted(removed))}")
        changes += len(removed)

    # Check changed properties on shared fields
    for field in sorted(old_fields & new_fields):
        of = old["models"][model]["fields"][field]
        nf = new["models"][model]["fields"][field]
        diffs = []
        for prop in ("type", "nullable", "default", "sensitive", "cast", "enum_values"):
            ov = of.get(prop)
            nv = nf.get(prop)
            if ov != nv:
                diffs.append(f"{prop}: {ov} -> {nv}")
        if diffs:
            print(f"  ~ {model}.{field}: {'; '.join(diffs)}")
            changes += 1

# Enums
old_enums = set(old.get("enums", {}).keys())
new_enums = set(new.get("enums", {}).keys())
added_enums = new_enums - old_enums
removed_enums = old_enums - new_enums
if added_enums:
    print(f"\n+ New enums: {', '.join(sorted(added_enums))}")
    changes += len(added_enums)
if removed_enums:
    print(f"\n- Removed enums: {', '.join(sorted(removed_enums))}")
    changes += len(removed_enums)

for enum in sorted(old_enums & new_enums):
    old_vals = set(old["enums"][enum])
    new_vals = set(new["enums"][enum])
    added = new_vals - old_vals
    removed = old_vals - new_vals
    if added or removed:
        if added:
            print(f"  + {enum}: new values: {', '.join(sorted(added))}")
        if removed:
            print(f"  - {enum}: removed values: {', '.join(sorted(removed))}")
        changes += 1

if changes == 0:
    print("No changes detected.")
else:
    print(f"\nTotal: {changes} change(s)")
PYTHON
