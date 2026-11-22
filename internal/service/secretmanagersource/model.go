package secretmanagersource

import (
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type settingFields struct {
	InfisicalProjectID   types.String `tfsdk:"infisical_project_id"`
	InfisicalEnvironment types.String `tfsdk:"infisical_environment"`
	InfisicalSecretPath  types.String `tfsdk:"infisical_secret_path"`
	DopplerProject       types.String `tfsdk:"doppler_project"`
	DopplerConfig        types.String `tfsdk:"doppler_config"`
	VaultMount           types.String `tfsdk:"vault_mount"`
	VaultPath            types.String `tfsdk:"vault_path"`
}

type model struct {
	ApplicationUUID      types.String `tfsdk:"application_uuid"`
	ServiceUUID          types.String `tfsdk:"service_uuid"`
	DatabaseUUID         types.String `tfsdk:"database_uuid"`
	IntegrationTokenUUID types.String `tfsdk:"integration_token_uuid"`
	IntegrationProvider  types.String `tfsdk:"integration_provider"`
	settingFields
}

type dataModel struct {
	ApplicationUUID      types.String `tfsdk:"application_uuid"`
	ServiceUUID          types.String `tfsdk:"service_uuid"`
	DatabaseUUID         types.String `tfsdk:"database_uuid"`
	Linked               types.Bool   `tfsdk:"linked"`
	IntegrationTokenUUID types.String `tfsdk:"integration_token_uuid"`
	IntegrationProvider  types.String `tfsdk:"integration_provider"`
	settingFields
}

func parentOf(app, svc, db types.String) (string, string, bool) {
	switch {
	case configuredString(app):
		return "applications", app.ValueString(), true
	case configuredString(svc):
		return "services", svc.ValueString(), true
	case configuredString(db):
		return "databases", db.ValueString(), true
	default:
		return "", "", false
	}
}

func parentCount(app, svc, db types.String) int {
	n := 0
	for _, v := range []types.String{app, svc, db} {
		if !v.IsNull() && !v.IsUnknown() {
			n++
		}
	}
	return n
}

func configuredString(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && strings.TrimSpace(v.ValueString()) != ""
}

func settingsBody(s settingFields) map[string]string {
	out := map[string]string{}
	add := func(key string, v types.String) {
		if configuredString(v) {
			out[key] = v.ValueString()
		}
	}
	add("project_id", s.InfisicalProjectID)
	add("environment", s.InfisicalEnvironment)
	add("secret_path", s.InfisicalSecretPath)
	add("project", s.DopplerProject)
	add("config", s.DopplerConfig)
	add("mount", s.VaultMount)
	add("path", s.VaultPath)
	return out
}

func clearSettings(s *settingFields) {
	s.InfisicalProjectID = types.StringNull()
	s.InfisicalEnvironment = types.StringNull()
	s.InfisicalSecretPath = types.StringNull()
	s.DopplerProject = types.StringNull()
	s.DopplerConfig = types.StringNull()
	s.VaultMount = types.StringNull()
	s.VaultPath = types.StringNull()
}

func lookupSetting(settings map[string]string, key string) types.String {
	if settings == nil {
		return types.StringNull()
	}
	v, ok := settings[key]
	if !ok || strings.TrimSpace(v) == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func knownProvider(provider string) bool {
	switch provider {
	case "doppler", "infisical", "vault":
		return true
	default:
		return false
	}
}

// storedMismatch reports a configured setting Coolify did not keep.
// A null attribute is unset and is not a mismatch. Import of a Doppler
// service token has null settings and null attributes, which matches.
func storedMismatch(s settingFields, settings map[string]string) (string, string) {
	if settings == nil {
		settings = map[string]string{}
	}
	pairs := []struct {
		attr string
		key  string
		val  types.String
	}{
		{"infisical_project_id", "project_id", s.InfisicalProjectID},
		{"infisical_environment", "environment", s.InfisicalEnvironment},
		{"infisical_secret_path", "secret_path", s.InfisicalSecretPath},
		{"doppler_project", "project", s.DopplerProject},
		{"doppler_config", "config", s.DopplerConfig},
		{"vault_mount", "mount", s.VaultMount},
		{"vault_path", "path", s.VaultPath},
	}
	for _, p := range pairs {
		if !configuredString(p.val) {
			continue
		}
		stored, ok := settings[p.key]
		if !ok {
			return p.attr, fmt.Sprintf("Coolify did not store %s. A Doppler service token (dp.st.) cannot keep project and config, and a token only keeps settings for its own provider. Remove the attribute, or link a token that stores it.", p.attr)
		}
		if stored != p.val.ValueString() {
			return p.attr, fmt.Sprintf("Coolify stored %s as %q.", p.attr, stored)
		}
	}
	return "", ""
}

func applySource(got *client.SecretManagerSource, token, provider *types.String, settings *settingFields) error {
	if got.IntegrationTokenUUID != nil && *got.IntegrationTokenUUID != "" {
		*token = types.StringValue(*got.IntegrationTokenUUID)
	}
	prov := ""
	if got.Provider != nil {
		prov = *got.Provider
	}
	if !knownProvider(prov) {
		return fmt.Errorf("unexpected secret manager provider %q", prov)
	}
	*provider = types.StringValue(prov)
	clearSettings(settings)
	switch prov {
	case "doppler":
		settings.DopplerProject = lookupSetting(got.Settings, "project")
		settings.DopplerConfig = lookupSetting(got.Settings, "config")
	case "infisical":
		settings.InfisicalProjectID = lookupSetting(got.Settings, "project_id")
		settings.InfisicalEnvironment = lookupSetting(got.Settings, "environment")
		settings.InfisicalSecretPath = lookupSetting(got.Settings, "secret_path")
	case "vault":
		settings.VaultMount = lookupSetting(got.Settings, "mount")
		settings.VaultPath = lookupSetting(got.Settings, "path")
	}
	return nil
}
