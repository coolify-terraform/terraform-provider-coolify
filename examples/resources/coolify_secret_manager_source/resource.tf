resource "coolify_secret_manager_source" "app" {
  application_uuid       = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
  integration_token_uuid = "11111111-1111-4111-8111-111111111111"

  infisical_project_id  = "00000000-0000-4000-8000-000000000001"
  infisical_environment = "prod"
  infisical_secret_path = "/app"
}

resource "coolify_environment_variable" "api_key" {
  application_uuid = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
  key              = "API_KEY"
  value            = "{{vault.API_KEY}}"

  depends_on = [coolify_secret_manager_source.app]
}

resource "coolify_secret_manager_source" "doppler_account" {
  service_uuid           = "ffffffff-ffff-4fff-8fff-ffffffffffff"
  integration_token_uuid = "33333333-3333-4333-8333-333333333333"
  doppler_project        = "website"
  doppler_config         = "production"
}

# A Doppler service token (dp.st.) stores no project or config.
resource "coolify_secret_manager_source" "doppler_token" {
  database_uuid          = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
  integration_token_uuid = "44444444-4444-4444-8444-444444444444"
}

resource "coolify_secret_manager_source" "vault" {
  application_uuid       = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
  integration_token_uuid = "22222222-2222-4222-8222-222222222222"
  vault_mount            = "secret"
  vault_path             = "my-app/production"
}
