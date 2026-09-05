data "coolify_application" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}

# is_consistent_container_name_enabled is computed from Coolify application
# settings. Use it to detect drift when a volume-backed app needs stop-then-start
# deploys instead of rolling updates.

# domain_port_overrides is a computed GET-only map of full domain URL to
# container port. It is null on Coolify <= v4.3.14 (requires >= v4.3.15).
# Output the whole map; do not index a key without a null check.
output "domain_port_overrides" {
  value = data.coolify_application.example.domain_port_overrides
}
