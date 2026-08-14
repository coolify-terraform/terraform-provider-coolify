package serversentinel

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
	ServerUUID                        types.String `tfsdk:"server_uuid"`
	IsSentinelEnabled                 types.Bool   `tfsdk:"is_sentinel_enabled"`
	IsMetricsEnabled                  types.Bool   `tfsdk:"is_metrics_enabled"`
	IsSentinelDebugEnabled            types.Bool   `tfsdk:"is_sentinel_debug_enabled"`
	SentinelToken                     types.String `tfsdk:"sentinel_token"`
	SentinelMetricsRefreshRateSeconds types.Int64  `tfsdk:"sentinel_metrics_refresh_rate_seconds"`
	SentinelMetricsHistoryDays        types.Int64  `tfsdk:"sentinel_metrics_history_days"`
	SentinelPushIntervalSeconds       types.Int64  `tfsdk:"sentinel_push_interval_seconds"`
	SentinelCustomURL                 types.String `tfsdk:"sentinel_custom_url"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_sentinel"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify Sentinel (host metrics agent) settings for a server. Available on Coolify >= v4.1.1. Destroy sets is_sentinel_enabled to false.",
		Attributes: map[string]schema.Attribute{
			"server_uuid":                           schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"is_sentinel_enabled":                   schema.BoolAttribute{Optional: true, Computed: true},
			"is_metrics_enabled":                    schema.BoolAttribute{Optional: true, Computed: true},
			"is_sentinel_debug_enabled":             schema.BoolAttribute{Optional: true, Computed: true},
			"sentinel_token":                        schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Preserved when GET omits it."},
			"sentinel_metrics_refresh_rate_seconds": schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_metrics_history_days":         schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_push_interval_seconds":        schema.Int64Attribute{Optional: true, Computed: true},
			"sentinel_custom_url":                   schema.StringAttribute{Optional: true},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func boolPtrKnown(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

func toInput(m model) client.ServerSentinel {
	in := client.ServerSentinel{
		IsSentinelEnabled:      boolPtrKnown(m.IsSentinelEnabled),
		IsMetricsEnabled:       boolPtrKnown(m.IsMetricsEnabled),
		IsSentinelDebugEnabled: boolPtrKnown(m.IsSentinelDebugEnabled),
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
	if s.SentinelToken != "" {
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
	flatten(got, &state)
	if got.SentinelToken == "" {
		state.SentinelToken = token
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
		resp.Diagnostics.AddError("Error disabling Sentinel", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
