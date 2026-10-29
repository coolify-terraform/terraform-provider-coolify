package serverlogdrain

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
	ServerUUID             types.String `tfsdk:"server_uuid"`
	NewRelicEnabled        types.Bool   `tfsdk:"is_logdrain_newrelic_enabled"`
	NewRelicLicenseKey     types.String `tfsdk:"logdrain_newrelic_license_key"`
	NewRelicBaseURI        types.String `tfsdk:"logdrain_newrelic_base_uri"`
	AxiomEnabled           types.Bool   `tfsdk:"is_logdrain_axiom_enabled"`
	AxiomDatasetName       types.String `tfsdk:"logdrain_axiom_dataset_name"`
	AxiomAPIKey            types.String `tfsdk:"logdrain_axiom_api_key"`
	AxiomDomain            types.String `tfsdk:"logdrain_axiom_domain"`
	CustomEnabled          types.Bool   `tfsdk:"is_logdrain_custom_enabled"`
	CustomConfig           types.String `tfsdk:"logdrain_custom_config"`
	CustomConfigParser     types.String `tfsdk:"logdrain_custom_config_parser"`
	CloudWatchEnabled      types.Bool   `tfsdk:"is_logdrain_cloudwatch_enabled"`
	CloudWatchGroup        types.String `tfsdk:"logdrain_cloudwatch_group"`
	CloudWatchRegion       types.String `tfsdk:"logdrain_cloudwatch_region"`
	CloudWatchStreamPrefix types.String `tfsdk:"logdrain_cloudwatch_stream_prefix"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_log_drain"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify server log drain settings (New Relic, Axiom, or custom). Requires Coolify >= v4.3.0. Destroy disables all drains. An omitted setting keeps the last applied value and does not show as known after apply.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"is_logdrain_newrelic_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"logdrain_newrelic_license_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "New Relic license key. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"logdrain_newrelic_base_uri": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "New Relic log API base URI. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"is_logdrain_axiom_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"logdrain_axiom_dataset_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Axiom dataset name. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"logdrain_axiom_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Axiom API key. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"logdrain_axiom_domain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Base domain of the Axiom edge deployment, for example `us-east-1.aws.edge.axiom.co` or `eu-central-1.aws.edge.axiom.co`. Requires Coolify >= v4.5.1. When this is omitted, Coolify uses `us-east-1.aws.edge.axiom.co`. On older instances the provider keeps the value in state and does not send it.",
			},
			"is_logdrain_custom_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"logdrain_custom_config": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Custom log drain configuration. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"logdrain_custom_config_parser": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Custom log drain config parser. Omitting the attribute later does not clear the stored Coolify value.",
			},
			"is_logdrain_cloudwatch_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Send container logs to Amazon CloudWatch. Requires Coolify >= v4.5.0. On older instances the provider keeps the value in state and does not send it.",
			},
			"logdrain_cloudwatch_region": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "CloudWatch region. Requires Coolify >= v4.5.0. On older instances the provider keeps the value in state and does not send it.",
			},
			"logdrain_cloudwatch_group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "CloudWatch log group. Requires Coolify >= v4.5.0. On older instances the provider keeps the value in state and does not send it.",
			},
			"logdrain_cloudwatch_stream_prefix": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Prefix for CloudWatch log stream names. Requires Coolify >= v4.5.0. On older instances the provider keeps the value in state and does not send it.",
			},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func toInput(m model) client.ServerLogDrains {
	return client.ServerLogDrains{
		IsNewRelicEnabled:      flex.BoolValueOrNull(m.NewRelicEnabled),
		NewRelicLicenseKey:     m.NewRelicLicenseKey.ValueString(),
		NewRelicBaseURI:        m.NewRelicBaseURI.ValueString(),
		IsAxiomEnabled:         flex.BoolValueOrNull(m.AxiomEnabled),
		AxiomDatasetName:       m.AxiomDatasetName.ValueString(),
		AxiomAPIKey:            m.AxiomAPIKey.ValueString(),
		AxiomDomain:            m.AxiomDomain.ValueString(),
		IsCustomEnabled:        flex.BoolValueOrNull(m.CustomEnabled),
		CustomConfig:           m.CustomConfig.ValueString(),
		CustomConfigParser:     m.CustomConfigParser.ValueString(),
		IsCloudWatchEnabled:    flex.BoolValueOrNull(m.CloudWatchEnabled),
		CloudWatchGroup:        m.CloudWatchGroup.ValueString(),
		CloudWatchRegion:       m.CloudWatchRegion.ValueString(),
		CloudWatchStreamPrefix: m.CloudWatchStreamPrefix.ValueString(),
	}
}

func flatten(s *client.ServerLogDrains, m *model) {
	if s.IsNewRelicEnabled != nil {
		m.NewRelicEnabled = types.BoolValue(*s.IsNewRelicEnabled)
	} else if m.NewRelicEnabled.IsUnknown() {
		m.NewRelicEnabled = types.BoolValue(false)
	}
	flex.SetStringSeedOrClear(&m.NewRelicLicenseKey, s.NewRelicLicenseKey)
	flex.SetStringSeedOrClear(&m.NewRelicBaseURI, s.NewRelicBaseURI)
	if s.IsAxiomEnabled != nil {
		m.AxiomEnabled = types.BoolValue(*s.IsAxiomEnabled)
	} else if m.AxiomEnabled.IsUnknown() {
		m.AxiomEnabled = types.BoolValue(false)
	}
	flex.SetStringSeedOrClear(&m.AxiomDatasetName, s.AxiomDatasetName)
	flex.SetStringSeedOrClear(&m.AxiomAPIKey, s.AxiomAPIKey)
	flex.SetStringPreserveEmpty(&m.AxiomDomain, s.AxiomDomain)
	if s.IsCustomEnabled != nil {
		m.CustomEnabled = types.BoolValue(*s.IsCustomEnabled)
	} else if m.CustomEnabled.IsUnknown() {
		m.CustomEnabled = types.BoolValue(false)
	}
	// Encrypted on Coolify 4.5. An empty GET must not wipe a stored config.
	flex.SetStringPreserveEmpty(&m.CustomConfig, s.CustomConfig)
	flex.SetStringSeedOrClear(&m.CustomConfigParser, s.CustomConfigParser)
	if s.IsCloudWatchEnabled != nil {
		m.CloudWatchEnabled = types.BoolValue(*s.IsCloudWatchEnabled)
	} else if m.CloudWatchEnabled.IsUnknown() {
		m.CloudWatchEnabled = types.BoolNull()
	}
	flex.SetStringPreserveEmpty(&m.CloudWatchGroup, s.CloudWatchGroup)
	flex.SetStringPreserveEmpty(&m.CloudWatchRegion, s.CloudWatchRegion)
	flex.SetStringPreserveEmpty(&m.CloudWatchStreamPrefix, s.CloudWatchStreamPrefix)
}

func warnUnsupportedCloudWatch(c *client.Client, plan model, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsCoolify45() {
		return
	}
	configured := (!plan.CloudWatchEnabled.IsNull() && !plan.CloudWatchEnabled.IsUnknown()) ||
		flex.StringValueConfigured(plan.CloudWatchGroup) ||
		flex.StringValueConfigured(plan.CloudWatchRegion) ||
		flex.StringValueConfigured(plan.CloudWatchStreamPrefix)
	if !configured {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write CloudWatch log drain settings",
		fmt.Sprintf(
			"This Coolify instance (%s) does not accept CloudWatch log drain fields (added in Coolify v4.5.0). "+
				"The provider will keep the values in Terraform state but will not send them. "+
				"Upgrade to Coolify v4.5.0 or later, or remove the CloudWatch attributes from configuration.",
			ver,
		),
	)
}

func warnUnsupportedAxiomDomain(c *client.Client, plan model, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsCoolify451() {
		return
	}
	if !flex.StringValueConfigured(plan.AxiomDomain) {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write logdrain_axiom_domain",
		fmt.Sprintf(
			"This Coolify instance (%s) does not accept logdrain_axiom_domain (added in Coolify v4.5.1). "+
				"The provider will keep the value in Terraform state but will not send it. "+
				"Upgrade to Coolify v4.5.1 or later, or remove logdrain_axiom_domain from configuration.",
			ver,
		),
	)
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnUnsupportedCloudWatch(r.client, plan, &resp.Diagnostics)
	warnUnsupportedAxiomDomain(r.client, plan, &resp.Diagnostics)
	got, err := r.client.UpdateServerLogDrains(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying server log drains", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetServerLogDrains(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server log drains", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnUnsupportedCloudWatch(r.client, plan, &resp.Diagnostics)
	warnUnsupportedAxiomDomain(r.client, plan, &resp.Diagnostics)
	got, err := r.client.UpdateServerLogDrains(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating server log drains", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	off := false
	disabled := client.ServerLogDrains{IsNewRelicEnabled: &off, IsAxiomEnabled: &off, IsCustomEnabled: &off}
	if _, err := r.client.UpdateServerLogDrains(ctx, state.ServerUUID.ValueString(), disabled); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling server log drains", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
