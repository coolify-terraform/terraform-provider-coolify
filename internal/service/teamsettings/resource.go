package teamsettings

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const typeName = "coolify_team_settings"

var (
	_ resource.Resource                = (*res)(nil)
	_ resource.ResourceWithConfigure   = (*res)(nil)
	_ resource.ResourceWithImportState = (*res)(nil)
)

type res struct{ client *client.Client }

type model struct {
	ID                     types.String `tfsdk:"id"`
	ScanImagesOnNewServers types.Bool   `tfsdk:"scan_images_on_new_servers"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_settings"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages settings for the Coolify team bound to the API token (`GET /teams/current`, `PATCH /team`). " +
			"The resource id is always `current`. " +
			"`scan_images_on_new_servers` requires Coolify >= v4.7.0. On older instances the provider keeps a configured value in state and does not send it. " +
			"Destroy sets that flag to false when the instance accepts it. Import with id `current`.",
		Attributes: map[string]schema.Attribute{
			"id": notificationcommon.IDAttribute(),
			"scan_images_on_new_servers": schema.BoolAttribute{
				MarkdownDescription: "Whether image scanning is turned on for servers added to the team from now on. " +
					"Coolify's stored default is false. Requires Coolify >= v4.7.0. " +
					"On older instances the provider keeps the value in state and does not send it.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": typeName})
	warnOld(r.client, plan.ScanImagesOnNewServers, &resp.Diagnostics)
	plan.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	input := client.UpdateCurrentTeamInput{ScanImagesOnNewServers: flex.BoolValueOrNull(plan.ScanImagesOnNewServers)}
	if _, err := r.client.UpdateCurrentTeam(ctx, input); err != nil {
		resp.Diagnostics.AddError("Error updating team settings", err.Error())
		return
	}
	if err := readInto(ctx, r.client, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading team settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": typeName})
	if err := readInto(ctx, r.client, &state); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading team settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": typeName})
	warnOld(r.client, plan.ScanImagesOnNewServers, &resp.Diagnostics)
	input := client.UpdateCurrentTeamInput{
		ScanImagesOnNewServers: flex.BoolIfChanged(plan.ScanImagesOnNewServers, state.ScanImagesOnNewServers),
	}
	if _, err := r.client.UpdateCurrentTeam(ctx, input); err != nil {
		resp.Diagnostics.AddError("Error updating team settings", err.Error())
		return
	}
	if err := readInto(ctx, r.client, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading team settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": typeName})
	if r.client == nil || !r.client.SupportsCoolify47() {
		return
	}
	off := false
	if _, err := r.client.UpdateCurrentTeam(ctx, client.UpdateCurrentTeamInput{ScanImagesOnNewServers: &off}); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling scan_images_on_new_servers", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, typeName)
}

func readInto(ctx context.Context, c *client.Client, m *model) error {
	got, err := c.GetCurrentTeam(ctx)
	if err != nil {
		return err
	}
	prior := m.ScanImagesOnNewServers
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	if got.ScanImagesOnNewServers != nil {
		m.ScanImagesOnNewServers = types.BoolValue(*got.ScanImagesOnNewServers)
		return nil
	}
	if c == nil || !c.SupportsCoolify47() {
		if !prior.IsNull() && !prior.IsUnknown() {
			m.ScanImagesOnNewServers = prior
			return nil
		}
	}
	m.ScanImagesOnNewServers = types.BoolNull()
	return nil
}

func warnOld(c *client.Client, v types.Bool, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsCoolify47() {
		return
	}
	if v.IsNull() || v.IsUnknown() {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write scan_images_on_new_servers",
		fmt.Sprintf("This Coolify instance (%s) does not accept scan_images_on_new_servers (added in Coolify v4.7.0). "+
			"The provider will keep the value in Terraform state but will not send it.", ver),
	)
}
