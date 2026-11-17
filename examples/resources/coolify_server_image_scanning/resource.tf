resource "coolify_server_image_scanning" "example" {
  server_uuid           = coolify_server.example.uuid
  is_image_scan_enabled = true

  # Coolify >= v4.7.3. Each value must be one of the scanner's fixed options.
  image_scan_memory_mb     = 2048
  image_scan_cpus          = 1
  image_scan_syft_timeout  = 600
  image_scan_grype_timeout = 600
}
