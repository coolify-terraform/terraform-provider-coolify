package secretmanagersource

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                   = (*secretManagerSourceDataSource)(nil)
	_ datasource.DataSourceWithConfigure      = (*secretManagerSourceDataSource)(nil)
	_ datasource.DataSourceWithValidateConfig = (*secretManagerSourceDataSource)(nil)
)

type secretManagerSourceDataSource struct {
	client *client.Client
}

func NewDataSource() datasource.DataSource { return &secretManagerSourceDataSource{} }

func (d *secretManagerSourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_manager_source"
}

func (d *secretManagerSourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	parent := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: description,
			Optional:            true,
			Validators:          []validator.String{validate.UUID()},
		}
	}
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the secret manager source of one Coolify application, service, or database. Requires Coolify >= v4.6.0. " +
			"A parent with no source returns linked = false. A missing parent is an error.",
		Attributes: map[string]schema.Attribute{
			"application_uuid":       parent("Application UUID. Exactly one of application_uuid, service_uuid, or database_uuid must be set."),
			"service_uuid":           parent("Compose service UUID from coolify_service. Exactly one parent UUID must be set."),
			"database_uuid":          parent("Database UUID. Exactly one parent UUID must be set."),
			"linked":                 schema.BoolAttribute{MarkdownDescription: "Whether the parent has a secret manager source.", Computed: true},
			"integration_token_uuid": computed("Linked token UUID. Null when linked is false."),
			"integration_provider":   computed("Provider of the linked token: doppler, infisical, or vault. Coolify's API field is provider. Null when linked is false."),
			"infisical_project_id":   computed("Infisical project id. Null when this source is not Infisical or has no source."),
			"infisical_environment":  computed("Infisical environment. Null when unset."),
			"infisical_secret_path":  computed("Infisical secret path. Null when omitted. Coolify reads / in that case."),
			"doppler_project":        computed("Doppler project. Null for a service token, which stores no settings."),
			"doppler_config":         computed("Doppler config. Null for a service token."),
			"vault_mount":            computed("Vault mount. Null when this source is not Vault."),
			"vault_path":             computed("Vault secret path. Null when this source is not Vault."),
		},
	}
}

func (d *secretManagerSourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *secretManagerSourceDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var cfg dataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateParents(cfg.ApplicationUUID, cfg.ServiceUUID, cfg.DatabaseUUID, &resp.Diagnostics)
}

func (d *secretManagerSourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state dataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !coolify46Ready(d.client, &resp.Diagnostics) {
		return
	}
	validateParents(state.ApplicationUUID, state.ServiceUUID, state.DatabaseUUID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceType, uuid, ok := oneParent(state.ApplicationUUID, state.ServiceUUID, state.DatabaseUUID, &resp.Diagnostics)
	if !ok {
		return
	}
	got, err := d.client.GetSecretManagerSource(ctx, resourceType, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading secret manager source", fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
		return
	}
	if !got.Linked {
		clearSettings(&state.settingFields)
		state.Linked = types.BoolValue(false)
		state.IntegrationTokenUUID = types.StringNull()
		state.IntegrationProvider = types.StringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	if err := applySource(got, &state.IntegrationTokenUUID, &state.IntegrationProvider, &state.settingFields); err != nil {
		resp.Diagnostics.AddError("Error reading secret manager source", err.Error())
		return
	}
	state.Linked = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
