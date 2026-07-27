data "coolify_vultr_plans" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr.uuid
}

output "vultr_plan_ids" {
  value = [for p in data.coolify_vultr_plans.all.plans : p.id]
}
