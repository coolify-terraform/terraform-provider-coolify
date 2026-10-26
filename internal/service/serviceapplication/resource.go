package serviceapplication

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/service"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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

const needs45 = "Requires Coolify >= v4.5.0. On older instances the provider keeps the value in state and does not send it."

var (
	_ resource.Resource                = &serviceApplicationResource{}
	_ resource.ResourceWithConfigure   = &serviceApplicationResource{}
	_ resource.ResourceWithImportState = &serviceApplicationResource{}
)

type serviceApplicationResource struct {
	client *client.Client
}

type model struct {
	ServiceUUID                types.String `tfsdk:"service_uuid"`
	UUID                       types.String `tfsdk:"uuid"`
	Name                       types.String `tfsdk:"name"`
	FQDN                       types.String `tfsdk:"fqdn"`
	Status                     types.String `tfsdk:"status"`
	URL                        types.String `tfsdk:"url"`
	NoindexDomains             types.List   `tfsdk:"noindex_domains"`
	HumanName                  types.String `tfsdk:"human_name"`
	Description                types.String `tfsdk:"description"`
	Image                      types.String `tfsdk:"image"`
	ExcludeFromStatus          types.Bool   `tfsdk:"exclude_from_status"`
	IsLogDrainEnabled          types.Bool   `tfsdk:"is_log_drain_enabled"`
	IsGzipEnabled              types.Bool   `tfsdk:"is_gzip_enabled"`
	IsStripPrefixEnabled       types.Bool   `tfsdk:"is_stripprefix_enabled"`
	IsForceHTTPSEnabled        types.Bool   `tfsdk:"is_force_https_enabled"`
	IsPermanentRedirectEnabled types.Bool   `tfsdk:"is_permanent_redirect_enabled"`
	TraefikCertificateResolver types.String `tfsdk:"traefik_certificate_resolver"`
	MaxRestartCount            types.Int64  `tfsdk:"max_restart_count"`
	ForceDomainOverride        types.Bool   `tfsdk:"force_domain_override"`
}

func NewResource() resource.Resource { return &serviceApplicationResource{} }

func (r *serviceApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_application"
}

func (r *serviceApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configures one application that already exists inside a `coolify_service` compose stack. " +
			"Coolify creates the container with the service. Destroy removes this object from Terraform state and does not delete the container or its URL.\n\n" +
			"Import with `service_uuid/uuid`.",
		Attributes: map[string]schema.Attribute{
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent service.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          validatorUUID(),
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the service application. Read it from `coolify_service_applications`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          validatorUUID(),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Compose service name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "FQDN Coolify stored for this application.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Runtime status.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Comma-separated public URLs. Same rules as `coolify_service` `urls`. Coolify may already have a URL; leaving this unset keeps that value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"noindex_domains": schema.ListAttribute{
				MarkdownDescription: "Domains that should send `X-Robots-Tag: noindex`. Entries that are not on this application are ignored by Coolify. Requires Coolify >= v4.3.0. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"human_name": schema.StringAttribute{
				MarkdownDescription: "Display name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image override.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"exclude_from_status": schema.BoolAttribute{
				MarkdownDescription: "Exclude this container from the service status rollup.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"is_log_drain_enabled": schema.BoolAttribute{
				MarkdownDescription: "Send this container's logs to the server log drain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"is_gzip_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable gzip for this application.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"is_stripprefix_enabled": schema.BoolAttribute{
				MarkdownDescription: "Strip the path prefix.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"is_force_https_enabled": schema.BoolAttribute{
				MarkdownDescription: "Redirect HTTP to HTTPS. Requires Coolify >= v4.3.8. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"is_permanent_redirect_enabled": schema.BoolAttribute{
				MarkdownDescription: "Use permanent redirects (301/308) instead of temporary redirects. " + needs45,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolUseState()},
			},
			"traefik_certificate_resolver": schema.StringAttribute{
				MarkdownDescription: "Traefik certificate resolver. Null uses the server setting. `none` uses no resolver. " + needs45,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"max_restart_count": schema.Int64Attribute{
				MarkdownDescription: "Maximum Docker restart count before Coolify stops the container. `0` disables the limit. Requires Coolify >= v4.3.19. v4.4-rc.1 does not accept it. On older instances the provider keeps the value in state and does not send it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64UseState()},
			},
			"force_domain_override": schema.BoolAttribute{
				MarkdownDescription: "When true, the PATCH query allows the URL even if it conflicts with another resource.",
				Optional:            true,
			},
		},
	}
}

func (r *serviceApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *serviceApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnVersionGated(r.client, plan, &resp.Diagnostics)
	got, err := r.client.GetServiceApplication(ctx, plan.ServiceUUID.ValueString(), plan.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Service application not found",
				fmt.Sprintf("Coolify has no application %s on service %s. Create the compose service first, then copy the application UUID from coolify_service_applications.", plan.UUID.ValueString(), plan.ServiceUUID.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Error reading service application", err.Error())
		return
	}
	input := prepareInput(r.client, buildInput(plan, model{}))
	if !inputEmpty(input) || boolValue(plan.ForceDomainOverride) {
		got, err = r.client.UpdateServiceApplication(ctx, plan.ServiceUUID.ValueString(), plan.UUID.ValueString(), input, boolValue(plan.ForceDomainOverride))
		if err != nil {
			resp.Diagnostics.AddError("Error updating service application", err.Error())
			return
		}
	}
	flatten(got, &plan, r.client)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "configured service application", map[string]interface{}{"uuid": plan.UUID.ValueString()})
}

func (r *serviceApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetServiceApplication(ctx, state.ServiceUUID.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service application", err.Error())
		return
	}
	flatten(got, &state, r.client)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warnVersionGated(r.client, plan, &resp.Diagnostics)
	input := prepareInput(r.client, buildInput(plan, state))
	var got *client.ServiceApplication
	var err error
	if inputEmpty(input) && !boolValue(plan.ForceDomainOverride) {
		got, err = r.client.GetServiceApplication(ctx, plan.ServiceUUID.ValueString(), plan.UUID.ValueString())
	} else {
		got, err = r.client.UpdateServiceApplication(ctx, plan.ServiceUUID.ValueString(), plan.UUID.ValueString(), input, boolValue(plan.ForceDomainOverride))
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating service application", err.Error())
		return
	}
	flatten(got, &plan, r.client)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceApplicationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *serviceApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceUUID, appUUID, ok := strings.Cut(req.ID, "/")
	if !ok || validate.ImportUUID(serviceUUID) != nil || validate.ImportUUID(appUUID) != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "Import id must be service_uuid/uuid, with both values as UUIDs.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_uuid"), serviceUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), appUUID)...)
}

func buildInput(plan, state model) client.UpdateServiceApplicationInput {
	var input client.UpdateServiceApplicationInput
	input.URL = flex.StringIfChanged(plan.URL, state.URL)
	input.HumanName = flex.StringIfChanged(plan.HumanName, state.HumanName)
	input.Description = flex.StringIfChanged(plan.Description, state.Description)
	input.Image = flex.StringIfChanged(plan.Image, state.Image)
	input.TraefikCertificateResolver = flex.StringIfChanged(plan.TraefikCertificateResolver, state.TraefikCertificateResolver)
	input.ExcludeFromStatus = flex.BoolIfChanged(plan.ExcludeFromStatus, state.ExcludeFromStatus)
	input.IsLogDrainEnabled = flex.BoolIfChanged(plan.IsLogDrainEnabled, state.IsLogDrainEnabled)
	input.IsGzipEnabled = flex.BoolIfChanged(plan.IsGzipEnabled, state.IsGzipEnabled)
	input.IsStripPrefixEnabled = flex.BoolIfChanged(plan.IsStripPrefixEnabled, state.IsStripPrefixEnabled)
	input.IsForceHTTPSEnabled = flex.BoolIfChanged(plan.IsForceHTTPSEnabled, state.IsForceHTTPSEnabled)
	input.IsPermanentRedirectEnabled = flex.BoolIfChanged(plan.IsPermanentRedirectEnabled, state.IsPermanentRedirectEnabled)
	input.MaxRestartCount = flex.Int64IfChanged(plan.MaxRestartCount, state.MaxRestartCount)
	if !plan.NoindexDomains.Equal(state.NoindexDomains) {
		input.NoindexDomains = stringListPtr(plan.NoindexDomains)
	}
	return input
}

func prepareInput(c *client.Client, input client.UpdateServiceApplicationInput) client.UpdateServiceApplicationInput {
	if c == nil {
		return input
	}
	if !c.SupportsCoolify45() {
		input.IsPermanentRedirectEnabled = nil
		input.TraefikCertificateResolver = nil
	}
	if !c.SupportsApplicationSettingsV43() {
		input.NoindexDomains = nil
	}
	if !c.SupportsServiceApplicationForceHTTPS() {
		input.IsForceHTTPSEnabled = nil
	}
	if !c.SupportsServiceApplicationMaxRestart() {
		input.MaxRestartCount = nil
	}
	return input
}

func inputEmpty(input client.UpdateServiceApplicationInput) bool {
	return input.URL == nil && input.NoindexDomains == nil && input.HumanName == nil && input.Description == nil &&
		input.Image == nil && input.ExcludeFromStatus == nil && input.IsLogDrainEnabled == nil && input.IsGzipEnabled == nil &&
		input.IsStripPrefixEnabled == nil && input.IsForceHTTPSEnabled == nil && input.IsPermanentRedirectEnabled == nil &&
		input.TraefikCertificateResolver == nil && input.MaxRestartCount == nil
}

func flatten(api *client.ServiceApplication, m *model, c *client.Client) {
	m.Name = types.StringValue(api.Name)
	m.FQDN = types.StringValue(api.FQDN)
	m.Status = types.StringValue(api.Status)
	apiURL := api.URL
	if apiURL == "" {
		apiURL = api.FQDN
	}
	switch {
	case apiURL == "" && m.URL.IsUnknown():
		m.URL = types.StringNull()
	case apiURL != "" && !m.URL.IsNull() && !m.URL.IsUnknown() && service.ServiceURLEquivalent(m.URL.ValueString(), apiURL):
		// Keep the configured URL when Coolify only differs by scheme, host case, or a stored port.
	case apiURL != "" && (m.URL.IsNull() || m.URL.IsUnknown() || !service.ServiceURLEquivalent(m.URL.ValueString(), apiURL)):
		m.URL = types.StringValue(api.URL)
		if api.URL == "" {
			m.URL = types.StringValue(api.FQDN)
		}
	}
	keepOrAPIString(&m.HumanName, api.HumanName)
	keepOrAPIString(&m.Description, api.Description)
	keepOrAPIString(&m.Image, api.Image)
	setBool(&m.ExcludeFromStatus, api.ExcludeFromStatus)
	setBool(&m.IsLogDrainEnabled, api.IsLogDrainEnabled)
	setBool(&m.IsGzipEnabled, api.IsGzipEnabled)
	setBool(&m.IsStripPrefixEnabled, api.IsStripPrefixEnabled)
	if !keepConfigured(c != nil && !c.SupportsServiceApplicationForceHTTPS(), m.IsForceHTTPSEnabled) {
		setBool(&m.IsForceHTTPSEnabled, api.IsForceHTTPSEnabled)
	}
	if !keepConfigured(c != nil && !c.SupportsCoolify45(), m.IsPermanentRedirectEnabled) {
		setBool(&m.IsPermanentRedirectEnabled, api.IsPermanentRedirectEnabled)
	}
	if !keepConfiguredString(c != nil && !c.SupportsCoolify45(), m.TraefikCertificateResolver) {
		if api.TraefikCertificateResolver != "" {
			m.TraefikCertificateResolver = types.StringValue(api.TraefikCertificateResolver)
		} else if m.TraefikCertificateResolver.IsUnknown() {
			m.TraefikCertificateResolver = types.StringNull()
		}
	}
	if !keepConfiguredInt(c != nil && !c.SupportsServiceApplicationMaxRestart(), m.MaxRestartCount) {
		if api.MaxRestartCount != nil {
			m.MaxRestartCount = types.Int64Value(*api.MaxRestartCount)
		} else if m.MaxRestartCount.IsUnknown() {
			m.MaxRestartCount = types.Int64Null()
		}
	}
	if !keepConfiguredList(c != nil && !c.SupportsApplicationSettingsV43(), m.NoindexDomains) {
		application.FlattenNoindexDomains(api.NoindexDomains, &m.NoindexDomains)
	}
}

func keepConfigured(withhold bool, v types.Bool) bool {
	return withhold && !v.IsNull() && !v.IsUnknown()
}

func keepConfiguredString(withhold bool, v types.String) bool {
	return withhold && !v.IsNull() && !v.IsUnknown()
}

func keepConfiguredInt(withhold bool, v types.Int64) bool {
	return withhold && !v.IsNull() && !v.IsUnknown()
}

func keepConfiguredList(withhold bool, v types.List) bool {
	return withhold && !v.IsNull() && !v.IsUnknown()
}

func warnVersionGated(c *client.Client, plan model, diags *diag.Diagnostics) {
	if c == nil || diags == nil {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	if !c.SupportsCoolify45() {
		if (!plan.IsPermanentRedirectEnabled.IsNull() && !plan.IsPermanentRedirectEnabled.IsUnknown()) ||
			(!plan.TraefikCertificateResolver.IsNull() && !plan.TraefikCertificateResolver.IsUnknown()) {
			diags.AddWarning(
				"Coolify version cannot write service application v4.5 settings",
				fmt.Sprintf("This Coolify instance (%s) is older than v4.5.0. is_permanent_redirect_enabled and traefik_certificate_resolver stay in state and are not sent.", ver),
			)
		}
	}
	if !c.SupportsApplicationSettingsV43() && !plan.NoindexDomains.IsNull() && !plan.NoindexDomains.IsUnknown() {
		diags.AddWarning(
			"Coolify version cannot write noindex_domains",
			fmt.Sprintf("This Coolify instance (%s) is older than v4.3.0. noindex_domains stays in state and is not sent.", ver),
		)
	}
	if !c.SupportsServiceApplicationForceHTTPS() && !plan.IsForceHTTPSEnabled.IsNull() && !plan.IsForceHTTPSEnabled.IsUnknown() {
		diags.AddWarning(
			"Coolify version cannot write is_force_https_enabled",
			fmt.Sprintf("This Coolify instance (%s) is older than v4.3.8. is_force_https_enabled stays in state and is not sent.", ver),
		)
	}
	if !c.SupportsServiceApplicationMaxRestart() && !plan.MaxRestartCount.IsNull() && !plan.MaxRestartCount.IsUnknown() {
		diags.AddWarning(
			"Coolify version cannot write max_restart_count",
			fmt.Sprintf("This Coolify instance (%s) is older than v4.3.19 or is v4.4-rc.1. max_restart_count stays in state and is not sent.", ver),
		)
	}
}

func keepOrAPIString(dst *types.String, api string) {
	if api != "" {
		*dst = types.StringValue(api)
		return
	}
	if dst.IsUnknown() {
		*dst = types.StringNull()
	}
}

func setBool(dst *types.Bool, src *bool) {
	if src != nil {
		*dst = types.BoolValue(*src)
	} else if dst.IsUnknown() {
		*dst = types.BoolNull()
	}
}

func stringListPtr(list types.List) *[]string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	out := make([]string, 0, len(list.Elements()))
	for _, el := range list.Elements() {
		s, ok := el.(types.String)
		if ok && !s.IsNull() && !s.IsUnknown() {
			out = append(out, s.ValueString())
		}
	}
	return &out
}

func boolValue(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}

func validatorUUID() []validator.String { return []validator.String{validate.UUID()} }

func boolUseState() planmodifier.Bool { return boolplanmodifier.UseStateForUnknown() }

func int64UseState() planmodifier.Int64 { return int64planmodifier.UseStateForUnknown() }
