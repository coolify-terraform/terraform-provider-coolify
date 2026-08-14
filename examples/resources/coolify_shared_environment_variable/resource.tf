resource "coolify_shared_environment_variable" "team" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}
