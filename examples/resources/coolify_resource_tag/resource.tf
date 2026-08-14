resource "coolify_tag" "frontend" {
  name = "frontend"
}

resource "coolify_resource_tag" "app" {
  resource_type = "application"
  resource_uuid = coolify_application.example.uuid
  tag_name      = coolify_tag.frontend.name
}
