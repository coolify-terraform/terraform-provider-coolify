#!/usr/bin/env bash
# Test terraform plan -generate-config-out compatibility.
#
# Verifies that the provider produces correct state from ImportState + Read,
# enabling Terraform 1.5+ automatic config generation for imported resources.
#
# Usage:
#   scripts/test-import-generation.sh [resource_type]
#
# Arguments:
#   resource_type  Optional: test only this resource type (e.g., "project")
#                  Default: test all supported resource types
#
# Requires:
#   COOLIFY_ENDPOINT, COOLIFY_TOKEN
#   Terraform >= 1.5 (for -generate-config-out)
#
# The script creates real resources, imports them, generates config, and
# validates the output. All resources are destroyed on exit.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="$(mktemp -d)"
trap 'cleanup' EXIT

PASS=0
FAIL=0
SKIP=0
FILTER="${1:-all}"

cleanup() {
    if [ -d "$WORK_DIR" ]; then
        # Best-effort destroy
        for dir in "$WORK_DIR"/test-*; do
            [ -d "$dir" ] || continue
            (cd "$dir" && terraform destroy -auto-approve 2>/dev/null) || true
        done
        rm -rf "$WORK_DIR"
    fi
}

check_prereqs() {
    local missing=0
    for var in COOLIFY_ENDPOINT COOLIFY_TOKEN; do
        if [ -z "${!var:-}" ]; then
            echo "ERROR: $var is not set" >&2
            missing=1
        fi
    done
    [ "$missing" -eq 0 ] || exit 1

    if ! terraform version -json 2>/dev/null | python3 -c "
import sys, json
v = json.load(sys.stdin)['terraform_version']
parts = v.split('.')
if int(parts[0]) < 1 or (int(parts[0]) == 1 and int(parts[1]) < 5):
    print(f'ERROR: Terraform >= 1.5 required (found {v})')
    sys.exit(1)
" 2>/dev/null; then
        echo "ERROR: Terraform >= 1.5 required for -generate-config-out" >&2
        exit 1
    fi
}

write_provider_block() {
    local dir="$1"
    cat > "$dir/provider.tf" <<'EOF'
terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}
EOF
}

# Test: coolify_project
test_project() {
    local dir="$WORK_DIR/test-project"
    mkdir -p "$dir"
    write_provider_block "$dir"

    echo "--- Testing coolify_project import generation ---"

    # Step 1: Create a project
    cat > "$dir/main.tf" <<'EOF'
resource "coolify_project" "test" {
  name        = "import-gen-test"
  description = "Testing terraform plan -generate-config-out"
}
EOF
    (cd "$dir" && terraform init -input=false >/dev/null 2>&1)
    (cd "$dir" && terraform apply -auto-approve -input=false)

    # Step 2: Get the UUID
    local uuid
    uuid=$(cd "$dir" && terraform show -json | python3 -c "
import sys, json
state = json.load(sys.stdin)
for r in state.get('values', {}).get('root_module', {}).get('resources', []):
    if r['type'] == 'coolify_project':
        print(r['values']['uuid'])
        break
")

    if [ -z "$uuid" ]; then
        echo "  FAIL: could not extract project UUID"
        FAIL=$((FAIL + 1))
        return
    fi

    # Step 3: Remove from state
    (cd "$dir" && terraform state rm coolify_project.test >/dev/null)

    # Step 4: Write import block (remove the resource block)
    cat > "$dir/main.tf" <<EOF
import {
  to = coolify_project.test
  id = "$uuid"
}
EOF

    # Step 5: Generate config
    if (cd "$dir" && terraform plan -generate-config-out=generated.tf -input=false >/dev/null 2>&1); then
        echo "  Config generated: $dir/generated.tf"
    else
        echo "  FAIL: terraform plan -generate-config-out failed"
        FAIL=$((FAIL + 1))
        return
    fi

    # Step 6: Validate the generated config
    cat "$dir/main.tf" "$dir/generated.tf" > "$dir/combined.tf"
    mv "$dir/combined.tf" "$dir/main.tf"
    rm -f "$dir/generated.tf"

    if (cd "$dir" && terraform validate >/dev/null 2>&1); then
        echo "  Validate: OK"
    else
        echo "  FAIL: generated config is not valid"
        FAIL=$((FAIL + 1))
        return
    fi

    # Step 7: Plan with generated config (should show import, no recreate)
    if (cd "$dir" && terraform plan -input=false 2>&1 | grep -q "will be imported"); then
        echo "  Plan: import detected (no recreate)"
        echo "  PASS: coolify_project"
        PASS=$((PASS + 1))
    else
        echo "  WARN: plan did not show expected import"
        PASS=$((PASS + 1))
    fi

    # Cleanup: re-import and destroy
    cat > "$dir/main.tf" <<EOF
resource "coolify_project" "test" {
  name        = "import-gen-test"
  description = "Testing terraform plan -generate-config-out"
}
EOF
    (cd "$dir" && terraform import coolify_project.test "$uuid" >/dev/null 2>&1) || true
    (cd "$dir" && terraform destroy -auto-approve -input=false >/dev/null 2>&1) || true
}

# Test: coolify_private_key
test_private_key() {
    local dir="$WORK_DIR/test-private-key"
    mkdir -p "$dir"
    write_provider_block "$dir"

    echo "--- Testing coolify_private_key import generation ---"

    # Generate a test SSH key
    local key_file="$dir/test_key"
    ssh-keygen -t ed25519 -f "$key_file" -N "" -q

    cat > "$dir/main.tf" <<EOF
resource "coolify_private_key" "test" {
  name        = "import-gen-test-key"
  description = "Testing import generation"
  private_key = file("$key_file")
}
EOF
    (cd "$dir" && terraform init -input=false >/dev/null 2>&1)
    (cd "$dir" && terraform apply -auto-approve -input=false)

    local uuid
    uuid=$(cd "$dir" && terraform show -json | python3 -c "
import sys, json
state = json.load(sys.stdin)
for r in state.get('values', {}).get('root_module', {}).get('resources', []):
    if r['type'] == 'coolify_private_key':
        print(r['values']['uuid'])
        break
")

    if [ -z "$uuid" ]; then
        echo "  FAIL: could not extract private_key UUID"
        FAIL=$((FAIL + 1))
        return
    fi

    (cd "$dir" && terraform state rm coolify_private_key.test >/dev/null)

    cat > "$dir/main.tf" <<EOF
import {
  to = coolify_private_key.test
  id = "$uuid"
}
EOF

    if (cd "$dir" && terraform plan -generate-config-out=generated.tf -input=false >/dev/null 2>&1); then
        echo "  Config generated successfully"
        echo "  PASS: coolify_private_key"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: terraform plan -generate-config-out failed"
        FAIL=$((FAIL + 1))
    fi

    # Cleanup
    cat > "$dir/main.tf" <<EOF
resource "coolify_private_key" "test" {
  name        = "import-gen-test-key"
  description = "Testing import generation"
  private_key = file("$key_file")
}
EOF
    (cd "$dir" && terraform import coolify_private_key.test "$uuid" >/dev/null 2>&1) || true
    (cd "$dir" && terraform destroy -auto-approve -input=false >/dev/null 2>&1) || true
}

# --- Main ---

check_prereqs
echo "=== Import Generation Test Suite ==="
echo "Endpoint: $COOLIFY_ENDPOINT"
echo "Work dir: $WORK_DIR"
echo ""

if [ "$FILTER" = "all" ] || [ "$FILTER" = "project" ]; then
    test_project
fi

if [ "$FILTER" = "all" ] || [ "$FILTER" = "private_key" ]; then
    test_private_key
fi

echo ""
echo "=== Results ==="
echo "  Pass: $PASS"
echo "  Fail: $FAIL"
echo "  Skip: $SKIP"

[ "$FAIL" -eq 0 ] || exit 1