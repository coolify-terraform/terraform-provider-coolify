provider "coolify" {
  endpoint = var.coolify_endpoint # Or set COOLIFY_ENDPOINT env var
  token    = var.coolify_token    # Or set COOLIFY_TOKEN env var

  # Optional: Cloudflare Access (when Coolify is behind CF Access)
  # cf_access_client_id     = var.cf_access_client_id     # Or set COOLIFY_CF_ACCESS_CLIENT_ID
  # cf_access_client_secret = var.cf_access_client_secret  # Or set COOLIFY_CF_ACCESS_CLIENT_SECRET

  # Optional: extra headers for an authenticating proxy, such as Google Cloud IAP.
  # Or set COOLIFY_HEADERS to a JSON object.
  # headers = {
  #   "Proxy-Authorization" = "Bearer ${var.iap_id_token}"
  # }

  # Optional: per-attempt HTTP timeout in seconds (default 30).
  # request_timeout = 60

  # Optional: mutual TLS. Set both, or neither.
  # client_cert = file("${path.module}/client.pem")
  # client_key  = file("${path.module}/client.key")

  # Optional: certificate name when it differs from the endpoint host.
  # tls_server_name = "coolify.internal.example"

  # Optional: outbound proxy for this machine. Not the proxy in front of Coolify.
  # proxy {
  #   url      = "http://proxy.example:8080"
  #   username = "ci"
  #   password = var.proxy_password
  # }
}