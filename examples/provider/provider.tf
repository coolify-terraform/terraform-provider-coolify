provider "coolify" {
  endpoint = var.coolify_endpoint # Or set COOLIFY_ENDPOINT env var
  token    = var.coolify_token    # Or set COOLIFY_TOKEN env var

  # Optional: Cloudflare Access (when Coolify is behind CF Access)
  # cf_access_client_id     = var.cf_access_client_id     # Or set COOLIFY_CF_ACCESS_CLIENT_ID
  # cf_access_client_secret = var.cf_access_client_secret  # Or set COOLIFY_CF_ACCESS_CLIENT_SECRET
}