resource "coolify_server" "example" {
  name             = "my-server"
  description      = "Production server"
  ip               = "192.168.1.100"
  port             = 22
  user             = "root"
  private_key_uuid = coolify_private_key.example.uuid

  # Optional server settings (uncomment as needed):
  # concurrent_builds                      = 2       # Max parallel deployments (default: 2)
  # dynamic_timeout                        = 3600    # Deployment timeout in seconds (default: 3600)
  # deployment_queue_limit                 = 25      # Max queued deployments, 0 = unlimited (default: 25)
  # server_disk_usage_notification_threshold = 80    # Disk usage alert threshold % (default: 80)
  # server_disk_usage_check_frequency      = "*/5 * * * *"  # Cron schedule for disk checks

  # Extended settings:
  # wildcard_domain      = "example.com"    # Wildcard domain for apps
  # server_timezone      = "America/New_York"
  # is_cloudflare_tunnel = false
  # is_metrics_enabled   = true
  # is_terminal_enabled  = true

  # Sentinel monitoring:
  # is_sentinel_enabled                 = true
  # sentinel_metrics_history_days       = 7
  # sentinel_metrics_refresh_rate_seconds = 10
  # sentinel_push_interval_seconds      = 60

  # Docker cleanup:
  # docker_cleanup_frequency  = "0 0 * * *"  # Daily at midnight
  # docker_cleanup_threshold  = 80            # % disk usage trigger
  # force_docker_cleanup      = false
  # delete_unused_volumes     = false
  # delete_unused_networks    = false
  # generate_exact_labels     = false
}