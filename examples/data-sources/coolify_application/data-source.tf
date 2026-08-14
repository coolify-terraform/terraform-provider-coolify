data "coolify_application" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}

# is_consistent_container_name_enabled is computed from Coolify application
# settings. Use it to detect drift when a volume-backed app needs stop-then-start
# deploys instead of rolling updates.
