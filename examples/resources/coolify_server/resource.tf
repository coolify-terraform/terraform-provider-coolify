resource "coolify_server" "example" {
  name             = "my-server"
  description      = "Production server"
  ip               = "192.168.1.100"
  port             = 22
  user             = "root"
  private_key_uuid = coolify_private_key.example.uuid

  # Optional server settings (uncomment as needed):
  # is_build_server                      = true    # Use this server for application builds
  # concurrent_builds                      = 2       # Max parallel deployments (default: 2)
  # dynamic_timeout                        = 3600    # Deployment timeout in seconds (default: 3600)
  # deployment_queue_limit                 = 25      # Max queued deployments (must be at least 1, default: 25)
  # connection_timeout                     = 10      # SSH connection timeout in seconds (1-300, default: 10)
  # server_disk_usage_notification_threshold = 80    # Disk usage alert threshold % (default: 80)
  # server_disk_usage_check_frequency      = "*/5 * * * *"  # Cron schedule for disk checks

  # Additional server settings such as wildcard domains, Sentinel flags,
  # and Docker cleanup options are exposed as read-only attributes because
  # the public server PATCH endpoint does not currently accept them.
}
