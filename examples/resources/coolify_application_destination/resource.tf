resource "coolify_application_destination" "extra" {
  application_uuid = coolify_application.example.uuid
  destination_uuid = coolify_destination.extra.uuid
}
