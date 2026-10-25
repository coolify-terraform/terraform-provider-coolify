# Turns the REST API and MCP server on.
# On Coolify 4.5.0 and later, enabled = true leaves the API unchanged.
resource "coolify_api_settings" "main" {
  enabled     = true
  mcp_enabled = true
}
