package apisettings

import (
	"context"
	"errors"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// errAPIEnableRemoved is returned when an apply would turn the API from
// off to on on a Coolify that no longer has POST /api/v1/enable.
// The text must not contain "403" or "not allowed": acceptance tests
// treat those as a non-root token skip.
var errAPIEnableRemoved = errors.New("Coolify 4.5.0 removed POST /api/v1/enable, so Terraform cannot turn the API on. Turn it back on in the Coolify UI, remove this resource from state, and apply again to record enabled = true. Set enabled = false to keep the current state")

var (
	_ resource.Resource              = (*apiSettingsResource)(nil)
	_ resource.ResourceWithConfigure = (*apiSettingsResource)(nil)
)

type apiSettingsResource struct {
	client *client.Client
}

type apiSettingsModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	MCPEnabled types.Bool `tfsdk:"mcp_enabled"`
}

func NewResource() resource.Resource {
	return &apiSettingsResource{}
}

func (r *apiSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_settings"
}

func (r *apiSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify instance-level settings: the REST API and the MCP server. Requires a root team (team 0) API token. On destroy, the MCP server is disabled. On Coolify before 4.5.0, destroy also re-enables the API. Coolify 4.5.0 removed POST /api/v1/enable, so destroy cannot turn the API back on.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Coolify REST API is enabled. On Coolify 4.5.0 and later, Terraform cannot turn the API on (Coolify removed `POST /api/v1/enable`). `true` leaves the API unchanged when it is already on. `false` still disables it. To record `true` again after it was `false`, turn the API on in the Coolify UI, remove this resource from state, and apply again. Destroy does not re-enable the API on those versions.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"mcp_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Coolify MCP server is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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

	if err := r.applyAPIState(ctx, plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring API settings", err.Error())
		return
	}
	if err := r.applyMCPState(ctx, plan.MCPEnabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring MCP settings", err.Error())
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

	var prior apiSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_api_settings"})

	if err := r.rejectAPIReenable(ctx, prior, plan); err != nil {
		resp.Diagnostics.AddError("Error configuring API settings", err.Error())
		return
	}
	if err := r.applyAPIState(ctx, plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring API settings", err.Error())
		return
	}
	if err := r.applyMCPState(ctx, plan.MCPEnabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error configuring MCP settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var prior apiSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource (restoring safe defaults)", map[string]interface{}{"resource_type": "coolify_api_settings"})
	r.restoreAPIOnDestroy(ctx, prior.Enabled.ValueBool(), resp)
	if err := r.client.DisableMCP(ctx); err != nil {
		resp.Diagnostics.AddWarning("Could not disable MCP server on destroy", err.Error())
	}
}

// rejectAPIReenable stops an update that would record the API as on when
// Coolify can no longer turn it on. Create is not checked. A false-to-true
// update is the case that would hide a still-disabled API: Coolify Cloud
// skips the API-enabled check, so Configure does not prove the flag.
func (r *apiSettingsResource) rejectAPIReenable(ctx context.Context, prior, plan apiSettingsModel) error {
	if !turningAPIOn(prior.Enabled, plan.Enabled) {
		return nil
	}
	available, err := r.client.APIEnableRouteAvailable(ctx)
	if err != nil {
		return fmt.Errorf("checking whether Coolify can enable the API: %w", err)
	}
	if available {
		return nil
	}
	return errAPIEnableRemoved
}

func turningAPIOn(prior, plan types.Bool) bool {
	if prior.IsNull() || prior.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		return false
	}
	return !prior.ValueBool() && plan.ValueBool()
}

func (r *apiSettingsResource) restoreAPIOnDestroy(ctx context.Context, wasEnabled bool, resp *resource.DeleteResponse) {
	available, err := r.client.APIEnableRouteAvailable(ctx)
	if err != nil {
		resp.Diagnostics.AddWarning("Could not re-enable API on destroy", err.Error())
		return
	}
	if !available {
		if !wasEnabled {
			resp.Diagnostics.AddWarning(
				"Could not re-enable API on destroy",
				"Coolify 4.5.0 removed POST /api/v1/enable, so the API stays off. Turn it back on in the Coolify UI.",
			)
		}
		return
	}
	if err := r.client.EnableAPI(ctx); err != nil {
		resp.Diagnostics.AddWarning("Could not re-enable API on destroy", err.Error())
	}
}

func (r *apiSettingsResource) applyAPIState(ctx context.Context, enabled bool) error {
	if !enabled {
		return r.client.DisableAPI(ctx)
	}
	available, err := r.client.APIEnableRouteAvailable(ctx)
	if err != nil {
		return err
	}
	if !available {
		return nil
	}
	return r.client.EnableAPI(ctx)
}

func (r *apiSettingsResource) applyMCPState(ctx context.Context, enabled bool) error {
	if enabled {
		return r.client.EnableMCP(ctx)
	}
	return r.client.DisableMCP(ctx)
}
