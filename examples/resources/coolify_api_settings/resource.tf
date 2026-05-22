# Enable the Coolify REST API and MCP server.
resource "coolify_api_settings" "main" {
  enabled     = true
  mcp_enabled = true
}
