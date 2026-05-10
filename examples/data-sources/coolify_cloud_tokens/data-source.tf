data "coolify_cloud_tokens" "all" {}

output "cloud_token_names" {
  value = [for t in data.coolify_cloud_tokens.all.cloud_tokens : t.name]
}
