data "coolify_tags" "all" {}

output "tag_names" {
  value = [for t in data.coolify_tags.all.tags : t.name]
}
