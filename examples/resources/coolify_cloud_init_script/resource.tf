resource "coolify_cloud_init_script" "bootstrap" {
  name   = "bootstrap"
  script = <<-EOT
    #cloud-config
    packages:
      - curl
  EOT
}
