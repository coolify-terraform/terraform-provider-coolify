package secretmanagersource

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateParents(app, svc, db types.String, diags *diag.Diagnostics) {
	for _, v := range []types.String{app, svc, db} {
		if v.IsUnknown() {
			return
		}
	}
	if parentCount(app, svc, db) != 1 {
		diags.AddError(
			"Exactly one parent",
			"Exactly one of application_uuid, service_uuid, or database_uuid must be set.",
		)
	}
}

func validateSettings(s settingFields, diags *diag.Diagnostics) {
	fields := settingList(s)
	for _, f := range fields {
		if f.val.IsUnknown() {
			return
		}
		if !f.val.IsNull() && strings.TrimSpace(f.val.ValueString()) == "" {
			diags.AddAttributeError(path.Root(f.attr), "Empty setting", "Settings cannot be empty or whitespace.")
		}
	}
	if diags.HasError() {
		return
	}
	infisical := groupSet(s.InfisicalProjectID, s.InfisicalEnvironment, s.InfisicalSecretPath)
	doppler := groupSet(s.DopplerProject, s.DopplerConfig)
	vault := groupSet(s.VaultMount, s.VaultPath)
	n := 0
	for _, on := range []bool{infisical, doppler, vault} {
		if on {
			n++
		}
	}
	if n > 1 {
		diags.AddError("One provider's settings", "Set settings for one provider only.")
		return
	}
	if infisical && (!configuredString(s.InfisicalProjectID) || !configuredString(s.InfisicalEnvironment)) {
		diags.AddError(
			"Incomplete Infisical settings",
			"infisical_project_id and infisical_environment are both required. infisical_secret_path is optional and cannot be set alone.",
		)
	}
	if doppler && (!configuredString(s.DopplerProject) || !configuredString(s.DopplerConfig)) {
		diags.AddError("Incomplete Doppler settings", "Set both doppler_project and doppler_config, or neither. A Doppler service token (dp.st.) must omit both.")
	}
	if vault && (!configuredString(s.VaultMount) || !configuredString(s.VaultPath)) {
		diags.AddError("Incomplete Vault settings", "Set both vault_mount and vault_path.")
	}
}

type namedString struct {
	attr string
	val  types.String
}

func settingList(s settingFields) []namedString {
	return []namedString{
		{"infisical_project_id", s.InfisicalProjectID},
		{"infisical_environment", s.InfisicalEnvironment},
		{"infisical_secret_path", s.InfisicalSecretPath},
		{"doppler_project", s.DopplerProject},
		{"doppler_config", s.DopplerConfig},
		{"vault_mount", s.VaultMount},
		{"vault_path", s.VaultPath},
	}
}

func groupSet(values ...types.String) bool {
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			return true
		}
	}
	return false
}
