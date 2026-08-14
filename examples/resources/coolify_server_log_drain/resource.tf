variable "axiom_api_key" {
  type      = string
  sensitive = true
}

resource "coolify_server_log_drain" "example" {
  server_uuid                 = coolify_server.example.uuid
  is_logdrain_axiom_enabled   = true
  logdrain_axiom_dataset_name = "coolify"
  logdrain_axiom_api_key      = var.axiom_api_key
}
