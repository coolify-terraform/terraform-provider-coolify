resource "coolify_cloud_token" "hostinger" {
  name           = "hostinger"
  cloud_provider = "hostinger"
  token          = var.hostinger_token
}

resource "coolify_server_hostinger" "app" {
  # Hostinger uses this as the VPS hostname only when it contains a dot.
  # A single label is stored on the Coolify server, and Hostinger picks the machine hostname.
  name                      = "app-node.example.com"
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
  item_id                   = "hostingercom-vps-kvm2"
  data_center_id            = 1
  template_id               = 7
  private_key_uuid          = coolify_private_key.main.uuid

  # public_key_ids and post_install_script_id are optional purchase fields.
  # Changing a purchase field replaces the Coolify server. It does not cancel
  # the previous Hostinger subscription. Destroy also leaves billing in hPanel.
  # public_key_ids         = [data.coolify_hostinger_ssh_keys.all.ssh_keys[0].id]
  # post_install_script_id = data.coolify_hostinger_post_install_scripts.all.scripts[0].id
}

variable "hostinger_token" {
  type      = string
  sensitive = true
}
