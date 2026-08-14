package serverproxy

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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*serverProxyResource)(nil)
	_ resource.ResourceWithConfigure   = (*serverProxyResource)(nil)
	_ resource.ResourceWithImportState = (*serverProxyResource)(nil)
)

type serverProxyResource struct{ client *client.Client }

type serverProxyModel struct {
	ServerUUID          types.String `tfsdk:"server_uuid"`
	RedirectEnabled     types.Bool   `tfsdk:"redirect_enabled"`
	RedirectURL         types.String `tfsdk:"redirect_url"`
	GenerateExactLabels types.Bool   `tfsdk:"generate_exact_labels"`
	ProxyType           types.String `tfsdk:"proxy_type"`
	Configuration       types.String `tfsdk:"configuration"`
}

func NewResource() resource.Resource { return &serverProxyResource{} }

func (r *serverProxyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_proxy"
}

func (r *serverProxyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify proxy settings and optional raw configuration for a server. Requires Coolify >= v4.3.0. Destroy leaves the remote proxy configuration in place.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{Required: true, MarkdownDescription: "Server UUID.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"redirect_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether HTTP to HTTPS redirect is enabled. Coolify defaults this to `true`. " +
					"Setting `false` is ignored by Coolify today (`$request->has('redirect_enabled')` treats JSON `false` as absent). Requires Coolify >= v4.3.0.",
			},
			"redirect_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "HTTPS redirect target URL. Coolify persists this field (`$request->exists('redirect_url')`). Use a resolvable host; reserved names such as `example.invalid` return 422.",
			},
			"generate_exact_labels": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether to generate exact Docker labels (removes extra labels from containers). " +
					"Setting `false` is ignored by Coolify today (`$request->has('generate_exact_labels')` treats JSON `false` as absent). Requires Coolify >= v4.3.0.",
			},
			"proxy_type":    schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Proxy type (for example traefik or caddy)."},
			"configuration": schema.StringAttribute{Optional: true, MarkdownDescription: "Raw proxy configuration written with PUT .../proxy/configuration."},
		},
	}
}

func (r *serverProxyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func proxyUpdateFromPlan(plan serverProxyModel) client.ServerProxyUpdateInput {
	in := client.ServerProxyUpdateInput{}
	if !plan.RedirectEnabled.IsNull() && !plan.RedirectEnabled.IsUnknown() {
		v := plan.RedirectEnabled.ValueBool()
		in.RedirectEnabled = &v
	}
	if !plan.RedirectURL.IsNull() && !plan.RedirectURL.IsUnknown() {
		v := plan.RedirectURL.ValueString()
		in.RedirectURL = &v
	}
	if !plan.GenerateExactLabels.IsNull() && !plan.GenerateExactLabels.IsUnknown() {
		v := plan.GenerateExactLabels.ValueBool()
		in.GenerateExactLabels = &v
	}
	if !plan.ProxyType.IsNull() && !plan.ProxyType.IsUnknown() {
		v := plan.ProxyType.ValueString()
		in.ProxyType = &v
	}
	return in
}

func flattenProxy(got *client.ServerProxy, plan *serverProxyModel) {
	if got.RedirectEnabled != nil {
		plan.RedirectEnabled = types.BoolValue(*got.RedirectEnabled)
	} else if plan.RedirectEnabled.IsUnknown() {
		plan.RedirectEnabled = types.BoolNull()
	}
	if got.RedirectURL != "" {
		plan.RedirectURL = types.StringValue(got.RedirectURL)
	} else if plan.RedirectURL.IsUnknown() {
		plan.RedirectURL = types.StringNull()
	}
	if got.GenerateExactLabels != nil {
		plan.GenerateExactLabels = types.BoolValue(*got.GenerateExactLabels)
	} else if plan.GenerateExactLabels.IsUnknown() {
		plan.GenerateExactLabels = types.BoolNull()
	}
	if got.ProxyType != "" {
		plan.ProxyType = types.StringValue(strings.ToLower(got.ProxyType))
	}
	if got.Configuration != "" && !plan.Configuration.IsNull() && !plan.Configuration.IsUnknown() {
		plan.Configuration = types.StringValue(got.Configuration)
	}
}

func (r *serverProxyResource) apply(ctx context.Context, plan *serverProxyModel) error {
	uuid := plan.ServerUUID.ValueString()
	got, err := r.client.UpdateServerProxy(ctx, uuid, proxyUpdateFromPlan(*plan))
	if err != nil {
		return err
	}
	flattenProxy(got, plan)
	if !plan.Configuration.IsNull() && !plan.Configuration.IsUnknown() && plan.Configuration.ValueString() != "" {
		return r.client.PutServerProxyConfiguration(ctx, uuid, plan.Configuration.ValueString())
	}
	return nil
}

func (r *serverProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error applying server proxy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetServerProxy(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server proxy", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
		return
	}
	flattenProxy(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating server proxy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No DELETE API; leave remote config in place.
}

func (r *serverProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
