package integrationtoken

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = (*integrationTokenResource)(nil)
	_ resource.ResourceWithConfigure      = (*integrationTokenResource)(nil)
	_ resource.ResourceWithImportState    = (*integrationTokenResource)(nil)
	_ resource.ResourceWithValidateConfig = (*integrationTokenResource)(nil)
)

var dopplerTokenPattern = regexp.MustCompile(`^dp\.(st|sa)\.`)

type integrationTokenResource struct {
	client *client.Client
}

type integrationTokenModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Name              types.String `tfsdk:"name"`
	Provider          types.String `tfsdk:"integration_provider"`
	Token             types.String `tfsdk:"token"`
	Capabilities      types.List   `tfsdk:"capabilities"`
	AutomaticDNS      types.Bool   `tfsdk:"automatic_dns"`
	ProxiedDNS        types.Bool   `tfsdk:"proxied_dns"`
	MetadataBaseURL   types.String `tfsdk:"metadata_base_url"`
	MetadataClientID  types.String `tfsdk:"metadata_client_id"`
	MetadataNamespace types.String `tfsdk:"metadata_namespace"`
	MetadataTimeout   types.Int64  `tfsdk:"metadata_timeout"`
	DNSZonesCount     types.Int64  `tfsdk:"dns_zones_count"`
	ZonesSyncedAt     types.String `tfsdk:"zones_synced_at"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func NewResource() resource.Resource { return &integrationTokenResource{} }

func (r *integrationTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_token"
}

func (r *integrationTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify integration token for Cloudflare, Doppler, Infisical, or Vault. Requires Coolify >= v4.6.0. Create checks the secret with that provider. The secret and metadata are not returned on read.",
		Attributes:          integrationTokenAttributes(),
	}
}

func integrationTokenAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "Integration token UUID.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the integration token. At most 255 characters.",
			Required:            true,
			Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
		},
		"integration_provider": schema.StringAttribute{
			MarkdownDescription: "Integration provider: `cloudflare`, `doppler`, `infisical`, or `vault`. Sent as the API field `provider`. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{stringvalidator.OneOf("cloudflare", "doppler", "infisical", "vault")},
		},
		"token": schema.StringAttribute{
			MarkdownDescription: "Provider secret. Coolify never returns it. After import, set it again before the next change that checks the secret. Doppler secrets must start with `dp.st.` or `dp.sa.`.",
			Required:            true,
			Sensitive:           true,
		},
		"capabilities": schema.ListAttribute{
			MarkdownDescription: "What the token may do. Cloudflare: `dns` and `tunnel` (`tunnel` also needs `dns`). Doppler, Infisical, and Vault: `secrets`. Omit this to let Coolify use `dns` for Cloudflare and `secrets` for the others.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			Validators: []validator.List{
				listvalidator.UniqueValues(),
				listvalidator.ValueStringsAre(stringvalidator.OneOf("dns", "tunnel", "secrets")),
			},
		},
		"automatic_dns": schema.BoolAttribute{
			MarkdownDescription: "Cloudflare only. Create DNS records when a domain is added. Coolify's default is true. Other providers reject this attribute.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"proxied_dns": schema.BoolAttribute{
			MarkdownDescription: "Cloudflare only. Create new DNS records proxied (orange cloud). Coolify's default is false. Other providers reject this attribute.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"metadata_base_url": schema.StringAttribute{
			MarkdownDescription: "Secret-manager API URL. Required for Infisical and Vault. Forbidden for Cloudflare. Not returned on read.",
			Optional:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"metadata_client_id": schema.StringAttribute{
			MarkdownDescription: "Infisical client id. Required for Infisical. Forbidden for Cloudflare. Not returned on read.",
			Optional:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"metadata_namespace": schema.StringAttribute{
			MarkdownDescription: "Vault namespace. Forbidden for Cloudflare. Not returned on read, so import does not load it. Remove the attribute to clear a namespace already stored in state.",
			Optional:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"metadata_timeout": schema.Int64Attribute{
			MarkdownDescription: "Secret-manager request timeout in seconds (1-300). Forbidden for Cloudflare. Not returned on read, so import does not load it. Remove the attribute to clear a timeout already stored in state.",
			Optional:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:          []validator.Int64{int64validator.Between(1, 300)},
		},
		"dns_zones_count": schema.Int64Attribute{
			MarkdownDescription: "Number of Cloudflare DNS zones Coolify could read. Computed. Empty for other providers.",
			Computed:            true,
		},
		"zones_synced_at": schema.StringAttribute{
			MarkdownDescription: "When Cloudflare zones were last read. Computed.",
			Computed:            true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "When Coolify created the token.",
			Computed:            true,
		},
		"updated_at": schema.StringAttribute{
			MarkdownDescription: "When Coolify last updated the token.",
			Computed:            true,
		},
	}
}

func (r *integrationTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *integrationTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg integrationTokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateIntegrationToken(cfg, &resp.Diagnostics)
}

func (r *integrationTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.requireCoolify46(&resp.Diagnostics) {
		return
	}
	created, err := r.client.CreateIntegrationToken(ctx, createInput(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating integration token", fmt.Sprintf("integration token %q: %s", plan.Name.ValueString(), err))
		return
	}
	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetIntegrationToken(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Integration token created but refresh failed", fmt.Sprintf("token %s: %s", created.UUID, err))
		return
	}
	flattenIntegrationToken(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_integration_token", "uuid": created.UUID})
}

func (r *integrationTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.requireCoolify46(&resp.Diagnostics) {
		return
	}
	got, err := r.client.GetIntegrationToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading integration token", fmt.Sprintf("token %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenIntegrationToken(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *integrationTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state integrationTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.requireCoolify46(&resp.Diagnostics) {
		return
	}
	input := updateInput(ctx, plan, state)
	got, err := r.client.UpdateIntegrationToken(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating integration token", fmt.Sprintf("token %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenIntegrationToken(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *integrationTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.requireCoolify46(&resp.Diagnostics) {
		return
	}
	if err := r.client.DeleteIntegrationToken(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting integration token", fmt.Sprintf("token %s: %s", state.UUID.ValueString(), err))
	}
}

func (r *integrationTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func (r *integrationTokenResource) requireCoolify46(diags *diag.Diagnostics) bool {
	return coolify46Ready(r.client, diags)
}

func coolify46Ready(c *client.Client, diags *diag.Diagnostics) bool {
	if c == nil || c.SupportsCoolify46() {
		return true
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddError(
		"Integration tokens need Coolify >= v4.6.0",
		fmt.Sprintf("This Coolify instance (%s) does not have the integration token API. Upgrade Coolify before using coolify_integration_token.", ver),
	)
	return false
}
