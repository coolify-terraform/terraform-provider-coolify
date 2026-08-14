package cloudinitscript

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*cloudInitScriptResource)(nil)
	_ resource.ResourceWithConfigure   = (*cloudInitScriptResource)(nil)
	_ resource.ResourceWithImportState = (*cloudInitScriptResource)(nil)
)

type cloudInitScriptResource struct{ client *client.Client }

type cloudInitScriptModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Script types.String `tfsdk:"script"`
}

func NewResource() resource.Resource { return &cloudInitScriptResource{} }

func (r *cloudInitScriptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_init_script"
}

func (r *cloudInitScriptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify team cloud-init script. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true, MarkdownDescription: "Script UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":   schema.StringAttribute{Required: true, MarkdownDescription: "Script name."},
			"script": schema.StringAttribute{Required: true, MarkdownDescription: "Cloud-init YAML. Coolify validates it as YAML."},
		},
	}
}

func (r *cloudInitScriptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func flatten(s *client.CloudInitScript, m *cloudInitScriptModel) {
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	if s.Script != "" {
		if m.Script.IsNull() || m.Script.IsUnknown() || strings.TrimSpace(s.Script) != strings.TrimSpace(m.Script.ValueString()) {
			m.Script = types.StringValue(s.Script)
		}
	}
}

func (r *cloudInitScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudInitScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateCloudInitScript(ctx, client.CloudInitScriptInput{Name: plan.Name.ValueString(), Script: plan.Script.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating cloud-init script", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudInitScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudInitScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetCloudInitScript(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cloud-init script", fmt.Sprintf("%s: %s", state.UUID.ValueString(), err))
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudInitScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudInitScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateCloudInitScript(ctx, plan.UUID.ValueString(), client.CloudInitScriptInput{Name: plan.Name.ValueString(), Script: plan.Script.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating cloud-init script", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudInitScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudInitScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCloudInitScript(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting cloud-init script", err.Error())
	}
}

func (r *cloudInitScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
