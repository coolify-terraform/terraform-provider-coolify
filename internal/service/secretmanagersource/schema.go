package secretmanagersource

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const resourceDescription = "Links a Coolify application, service, or database to one Doppler, Infisical, or Vault integration token. Requires Coolify >= v4.6.0. " +
	"There is one source per parent. Create and update send PATCH, then read the source back. " +
	"Destroy removes the link and leaves environment variables and the integration token in place. " +
	"Put {{vault.KEY}} references on coolify_environment_variable. " +
	"Changing the parent forces a new resource. Changing the token or settings updates in place."

func parentResourceAttr(description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description,
		Optional:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators:          []validator.String{validate.UUID()},
	}
}

func settingAttr(description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description,
		Optional:            true,
	}
}

func resourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"application_uuid": parentResourceAttr("Application UUID. Exactly one of application_uuid, service_uuid, or database_uuid must be set. Changing this forces a new resource. A service application UUID is not a parent."),
		"service_uuid":     parentResourceAttr("Compose service UUID from coolify_service. Exactly one parent UUID must be set. Changing this forces a new resource."),
		"database_uuid":    parentResourceAttr("Database UUID. Exactly one parent UUID must be set. Changing this forces a new resource. The API path is /databases/{uuid}/secret-manager for every database type."),
		"integration_token_uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of a Doppler, Infisical, or Vault token from coolify_integration_token. The token must have the secrets capability. Updated in place.",
			Required:            true,
			Validators:          []validator.String{validate.UUID()},
		},
		"integration_provider": schema.StringAttribute{
			MarkdownDescription: "Provider of the linked token: doppler, infisical, or vault. Coolify's API field is provider. Computed. Stays in the plan while integration_token_uuid is unchanged.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{keepProviderUntilTokenChanges{}},
		},
		"infisical_project_id":  settingAttr("Infisical project id. Required with infisical_environment. Do not set Doppler or Vault settings on the same resource."),
		"infisical_environment": settingAttr("Infisical environment slug, such as prod. Required with infisical_project_id."),
		"infisical_secret_path": settingAttr("Infisical secret path. Optional. Coolify reads / when this is omitted. Setting / stores / and is a different value."),
		"doppler_project":       settingAttr("Doppler project. Required with doppler_config for a service account token (dp.sa.). Omit both for a service token (dp.st.). If Coolify does not store this, apply fails instead of clearing it."),
		"doppler_config":        settingAttr("Doppler config, such as production. Required with doppler_project for a service account token."),
		"vault_mount":           settingAttr("Vault mount. Required with vault_path. Coolify's fetch default of secret does not apply on write."),
		"vault_path":            settingAttr("Vault secret path. Required with vault_mount."),
	}
}
