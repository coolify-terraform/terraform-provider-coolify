package apisettings

import (
	"context"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = (*apiSettingsResource)(nil)
	_ resource.ResourceWithConfigure = (*apiSettingsResource)(nil)
)

type apiSettingsResource struct {
	client *client.Client
}

type apiSettingsModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func NewResource() resource.Resource {
	return &apiSettingsResource{}
}

func (r *apiSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_settings"
}

func (r *apiSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Coolify REST API enabled/disabled state. Requires a root team (team 0) API token. On destroy, the API is always re-enabled to prevent lockout.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Coolify REST API is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *apiSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *apiSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_api_settings"})

	if err := r.applyState(ctx, plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring API settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiSettingsResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// No GET endpoint exists for API settings state. Preserve state from last write.
}

func (r *apiSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_api_settings"})

	if err := r.applyState(ctx, plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring API settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Always re-enable the API on destroy to prevent lockout.
	tflog.Debug(ctx, "deleting resource (re-enabling API)", map[string]interface{}{"resource_type": "coolify_api_settings"})
	if err := r.client.EnableAPI(ctx); err != nil {
		resp.Diagnostics.AddWarning("Could not re-enable API on destroy", err.Error())
	}
}

func (r *apiSettingsResource) applyState(ctx context.Context, enabled bool) error {
	if enabled {
		return r.client.EnableAPI(ctx)
	}
	return r.client.DisableAPI(ctx)
}
