package serversentinel

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	ServerUUID                        types.String `tfsdk:"server_uuid"`
	IsSentinelEnabled                 types.Bool   `tfsdk:"is_sentinel_enabled"`
	IsMetricsEnabled                  types.Bool   `tfsdk:"is_metrics_enabled"`
	IsSentinelDebugEnabled            types.Bool   `tfsdk:"is_sentinel_debug_enabled"`
	SentinelToken                     types.String `tfsdk:"sentinel_token"`
	SentinelMetricsRefreshRateSeconds types.Int64  `tfsdk:"sentinel_metrics_refresh_rate_seconds"`
	SentinelMetricsHistoryDays        types.Int64  `tfsdk:"sentinel_metrics_history_days"`
	SentinelPushIntervalSeconds       types.Int64  `tfsdk:"sentinel_push_interval_seconds"`
	SentinelCustomURL                 types.String `tfsdk:"sentinel_custom_url"`
	TrafficTopN                       types.Int64  `tfsdk:"traffic_topn"`
	TrafficSampleThreshold            types.Int64  `tfsdk:"traffic_sample_threshold"`
	TrafficRetention1hDays            types.Int64  `tfsdk:"traffic_retention_1h_days"`
	TrafficRetention1dDays            types.Int64  `tfsdk:"traffic_retention_1d_days"`
	IsGeoIPEnabled                    types.Bool   `tfsdk:"is_geoip_enabled"`
	GeoIPRefreshDays                  types.Int64  `tfsdk:"geoip_refresh_days"`
	GeoIPMaxMindLicenseKey            types.String `tfsdk:"geoip_maxmind_license_key"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_sentinel"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify Sentinel (host metrics agent) settings for a server. Requires Coolify >= v4.3.0 (GET/PATCH `/servers/{uuid}/sentinel`). Destroy tries to set is_sentinel_enabled to false; newer Coolify extra-key 422s that field (Sentinel is mandatory on regular servers) and destroy then only drops state.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"is_sentinel_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether Sentinel is enabled. Coolify tip after 2026-09-15 " +
					"treats Sentinel as mandatory on regular servers, so PATCH extra-key 422s " +
					"this field (GET remains). Writable on Coolify 4.3.x before that change.",
			},
			"is_metrics_enabled":        schema.BoolAttribute{Optional: true, Computed: true},
			"is_sentinel_debug_enabled": schema.BoolAttribute{Optional: true, Computed: true},
			"sentinel_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Sentinel agent token. Preserved on refresh when GET omits it. After import, keep the token in configuration; GET typically hides it so import cannot seed state.",
			},
			"sentinel_metrics_refresh_rate_seconds": schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_metrics_history_days":         schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_push_interval_seconds":        schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_custom_url": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Custom Sentinel push URL. Writable on GET/PATCH `/servers/{uuid}/sentinel` " +
					"(unlike the read-only copy on `coolify_server`). Max 255 characters. " +
					"Seeded from GET when Coolify returns a non-empty value; keep it in configuration " +
					"after import if GET still omits it.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"traffic_topn": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "How many top talkers Sentinel stores for traffic analytics. Coolify default is `50`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
			},
			"traffic_sample_threshold": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Minimum bytes before a flow is sampled. Coolify default is `0`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.AtLeast(0)},
			},
			"traffic_retention_1h_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Days to retain 1-hour traffic buckets. Coolify default is `30`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
			},
			"traffic_retention_1d_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Days to retain 1-day traffic buckets. Coolify default is `395`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
			},
			"is_geoip_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether GeoIP lookup is enabled for Sentinel traffic. Coolify default is `true`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"geoip_refresh_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "How often Sentinel refreshes the GeoIP database, in days. Coolify default is `30`. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
			},
			"geoip_maxmind_license_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "MaxMind license key for GeoIP database downloads. Preserved on refresh when GET omits it. " +
					"Requires Coolify >= 4.4 (not in tag v4.3.23 or v4.4-rc.1). Against older instances the provider omits it on write. " +
					"After import, keep the key in configuration; GET typically hides it unless the token has read:sensitive.",
			},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func toInput(m model) client.ServerSentinel {
	in := client.ServerSentinel{
		IsSentinelEnabled:      flex.BoolValueOrNull(m.IsSentinelEnabled),
		IsMetricsEnabled:       flex.BoolValueOrNull(m.IsMetricsEnabled),
		IsSentinelDebugEnabled: flex.BoolValueOrNull(m.IsSentinelDebugEnabled),
		SentinelToken:          m.SentinelToken.ValueString(),
		SentinelCustomURL:      m.SentinelCustomURL.ValueString(),
	}
	if !m.SentinelMetricsRefreshRateSeconds.IsNull() && !m.SentinelMetricsRefreshRateSeconds.IsUnknown() {
		v := m.SentinelMetricsRefreshRateSeconds.ValueInt64()
		in.SentinelMetricsRefreshRateSeconds = &v
	}
	if !m.SentinelMetricsHistoryDays.IsNull() && !m.SentinelMetricsHistoryDays.IsUnknown() {
		v := m.SentinelMetricsHistoryDays.ValueInt64()
		in.SentinelMetricsHistoryDays = &v
	}
	if !m.SentinelPushIntervalSeconds.IsNull() && !m.SentinelPushIntervalSeconds.IsUnknown() {
		v := m.SentinelPushIntervalSeconds.ValueInt64()
		in.SentinelPushIntervalSeconds = &v
	}
	if !m.TrafficTopN.IsNull() && !m.TrafficTopN.IsUnknown() {
		v := m.TrafficTopN.ValueInt64()
		in.TrafficTopN = &v
	}
	if !m.TrafficSampleThreshold.IsNull() && !m.TrafficSampleThreshold.IsUnknown() {
		v := m.TrafficSampleThreshold.ValueInt64()
		in.TrafficSampleThreshold = &v
	}
	if !m.TrafficRetention1hDays.IsNull() && !m.TrafficRetention1hDays.IsUnknown() {
		v := m.TrafficRetention1hDays.ValueInt64()
		in.TrafficRetention1hDays = &v
	}
	if !m.TrafficRetention1dDays.IsNull() && !m.TrafficRetention1dDays.IsUnknown() {
		v := m.TrafficRetention1dDays.ValueInt64()
		in.TrafficRetention1dDays = &v
	}
	in.IsGeoIPEnabled = flex.BoolValueOrNull(m.IsGeoIPEnabled)
	if !m.GeoIPRefreshDays.IsNull() && !m.GeoIPRefreshDays.IsUnknown() {
		v := m.GeoIPRefreshDays.ValueInt64()
		in.GeoIPRefreshDays = &v
	}
	in.GeoIPMaxMindLicenseKey = m.GeoIPMaxMindLicenseKey.ValueString()
	return in
}

func flatten(s *client.ServerSentinel, m *model) {
	if s.IsSentinelEnabled != nil {
		m.IsSentinelEnabled = types.BoolValue(*s.IsSentinelEnabled)
	} else if m.IsSentinelEnabled.IsUnknown() {
		m.IsSentinelEnabled = types.BoolValue(false)
	}
	if s.IsMetricsEnabled != nil {
		m.IsMetricsEnabled = types.BoolValue(*s.IsMetricsEnabled)
	} else if m.IsMetricsEnabled.IsUnknown() {
		m.IsMetricsEnabled = types.BoolValue(false)
	}
	if s.IsSentinelDebugEnabled != nil {
		m.IsSentinelDebugEnabled = types.BoolValue(*s.IsSentinelDebugEnabled)
	} else if m.IsSentinelDebugEnabled.IsUnknown() {
		m.IsSentinelDebugEnabled = types.BoolValue(false)
	}
	if s.SentinelToken != "" && !m.SentinelToken.IsNull() {
		m.SentinelToken = types.StringValue(s.SentinelToken)
	}
	if s.SentinelMetricsRefreshRateSeconds != nil {
		m.SentinelMetricsRefreshRateSeconds = types.Int64Value(*s.SentinelMetricsRefreshRateSeconds)
	} else if m.SentinelMetricsRefreshRateSeconds.IsUnknown() {
		m.SentinelMetricsRefreshRateSeconds = types.Int64Null()
	}
	if s.SentinelMetricsHistoryDays != nil {
		m.SentinelMetricsHistoryDays = types.Int64Value(*s.SentinelMetricsHistoryDays)
	} else if m.SentinelMetricsHistoryDays.IsUnknown() {
		m.SentinelMetricsHistoryDays = types.Int64Null()
	}
	if s.SentinelPushIntervalSeconds != nil {
		m.SentinelPushIntervalSeconds = types.Int64Value(*s.SentinelPushIntervalSeconds)
	} else if m.SentinelPushIntervalSeconds.IsUnknown() {
		m.SentinelPushIntervalSeconds = types.Int64Null()
	}
	if s.SentinelCustomURL != "" {
		m.SentinelCustomURL = types.StringValue(s.SentinelCustomURL)
	} else if m.SentinelCustomURL.IsUnknown() {
		m.SentinelCustomURL = types.StringNull()
	}
	flattenInt64(s.TrafficTopN, &m.TrafficTopN)
	flattenInt64(s.TrafficSampleThreshold, &m.TrafficSampleThreshold)
	flattenInt64(s.TrafficRetention1hDays, &m.TrafficRetention1hDays)
	flattenInt64(s.TrafficRetention1dDays, &m.TrafficRetention1dDays)
	if s.IsGeoIPEnabled != nil {
		m.IsGeoIPEnabled = types.BoolValue(*s.IsGeoIPEnabled)
	} else if m.IsGeoIPEnabled.IsUnknown() {
		m.IsGeoIPEnabled = types.BoolNull()
	}
	flattenInt64(s.GeoIPRefreshDays, &m.GeoIPRefreshDays)
	if s.GeoIPMaxMindLicenseKey != "" && !m.GeoIPMaxMindLicenseKey.IsNull() {
		m.GeoIPMaxMindLicenseKey = types.StringValue(s.GeoIPMaxMindLicenseKey)
	}
}

func flattenInt64(src *int64, dst *types.Int64) {
	if src != nil {
		*dst = types.Int64Value(*src)
	} else if dst.IsUnknown() {
		*dst = types.Int64Null()
	}
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerSentinel(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying Sentinel settings", err.Error())
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
	token := state.SentinelToken
	got, err := r.client.GetServerSentinel(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Sentinel settings", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
		return
	}
	license := state.GeoIPMaxMindLicenseKey
	flatten(got, &state)
	if got.SentinelToken == "" {
		state.SentinelToken = token
	}
	if got.GeoIPMaxMindLicenseKey == "" {
		state.GeoIPMaxMindLicenseKey = license
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerSentinel(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Sentinel settings", err.Error())
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
	in := client.ServerSentinel{IsSentinelEnabled: &off}
	if _, err := r.client.UpdateServerSentinel(ctx, state.ServerUUID.ValueString(), in); err != nil && !client.IsNotFound(err) {
		// Newer Coolify extra-key 422s the enable flag and GET-computes it.
		// Destroy cannot disable remotely; drop state only.
		if !client.IsSentinelEnabledNotAllowed(err) {
			resp.Diagnostics.AddError("Error disabling Sentinel", err.Error())
		}
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
