resource "coolify_team_settings" "example" {
  # Turns image scanning on for servers added to this team from now on.
  # Requires Coolify >= v4.7.0. Older instances keep the value and do not send it.
  scan_images_on_new_servers = true
}
