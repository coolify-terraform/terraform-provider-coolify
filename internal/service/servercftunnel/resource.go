package servercftunnel

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*res)(nil)
	_ resource.ResourceWithConfigure   = (*res)(nil)
	_ resource.ResourceWithImportState = (*res)(nil)
)

type res struct{ client *client.Client }

type model struct {
	ServerUUID         types.String `tfsdk:"server_uuid"`
	IsCloudflareTunnel types.Bool   `tfsdk:"is_cloudflare_tunnel"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_cloudflare_tunnel"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages whether a Coolify server uses a Cloudflare tunnel. Requires Coolify >= v4.3.0. Destroy sets is_cloudflare_tunnel to false.",
		Attributes: map[string]schema.Attribute{
			"server_uuid":          schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"is_cloudflare_tunnel": schema.BoolAttribute{Required: true, MarkdownDescription: "Whether Cloudflare tunnel is enabled on the server."},
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
	got, err := r.client.UpdateServerCloudflareTunnel(ctx, plan.ServerUUID.ValueString(), plan.IsCloudflareTunnel.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error applying Cloudflare tunnel setting", err.Error())
		return
	}
	plan.IsCloudflareTunnel = types.BoolValue(got.IsCloudflareTunnel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetServerCloudflareTunnel(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Cloudflare tunnel setting", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
		return
	}
	state.IsCloudflareTunnel = types.BoolValue(got.IsCloudflareTunnel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerCloudflareTunnel(ctx, plan.ServerUUID.ValueString(), plan.IsCloudflareTunnel.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error updating Cloudflare tunnel setting", err.Error())
		return
	}
	plan.IsCloudflareTunnel = types.BoolValue(got.IsCloudflareTunnel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.UpdateServerCloudflareTunnel(ctx, state.ServerUUID.ValueString(), false); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling Cloudflare tunnel", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
