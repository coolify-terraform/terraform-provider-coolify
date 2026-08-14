package server

import (
	"context"
	"fmt"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithConfigure   = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
)

type serverResource struct {
	client *client.Client
}

type serverResourceModel struct {
	Timeouts                             timeouts.Value `tfsdk:"timeouts"`
	UUID                                 types.String   `tfsdk:"uuid"`
	Name                                 types.String   `tfsdk:"name"`
	Description                          types.String   `tfsdk:"description"`
	IP                                   types.String   `tfsdk:"ip"`
	Port                                 types.Int64    `tfsdk:"port"`
	User                                 types.String   `tfsdk:"user"`
	PrivateKeyUUID                       types.String   `tfsdk:"private_key_uuid"`
	IsBuildServer                        types.Bool     `tfsdk:"is_build_server"`
	IsReachable                          types.Bool     `tfsdk:"is_reachable"`
	IsUsable                             types.Bool     `tfsdk:"is_usable"`
	ConcurrentBuilds                     types.Int64    `tfsdk:"concurrent_builds"`
	DynamicTimeout                       types.Int64    `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                 types.Int64    `tfsdk:"deployment_queue_limit"`
	ConnectionTimeout                    types.Int64    `tfsdk:"connection_timeout"`
	ServerDiskUsageNotificationThreshold types.Int64    `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        types.String   `tfsdk:"server_disk_usage_check_frequency"`
	// Read-only extended settings returned by GET responses.
	WildcardDomain                    types.String `tfsdk:"wildcard_domain"`
	IsCloudFlareTunnel                types.Bool   `tfsdk:"is_cloudflare_tunnel"`
	ServerTimezone                    types.String `tfsdk:"server_timezone"`
	IsMetricsEnabled                  types.Bool   `tfsdk:"is_metrics_enabled"`
	IsTerminalEnabled                 types.Bool   `tfsdk:"is_terminal_enabled"`
	IsSentinelEnabled                 types.Bool   `tfsdk:"is_sentinel_enabled"`
	SentinelMetricsHistoryDays        types.Int64  `tfsdk:"sentinel_metrics_history_days"`
	SentinelMetricsRefreshRateSeconds types.Int64  `tfsdk:"sentinel_metrics_refresh_rate_seconds"`
	SentinelPushIntervalSeconds       types.Int64  `tfsdk:"sentinel_push_interval_seconds"`
	DockerCleanupFrequency            types.String `tfsdk:"docker_cleanup_frequency"`
	DockerCleanupThreshold            types.Int64  `tfsdk:"docker_cleanup_threshold"`
	ForceDockerCleanup                types.Bool   `tfsdk:"force_docker_cleanup"`
	DeleteUnusedVolumes               types.Bool   `tfsdk:"delete_unused_volumes"`
	DeleteUnusedNetworks              types.Bool   `tfsdk:"delete_unused_networks"`
	GenerateExactLabels               types.Bool   `tfsdk:"generate_exact_labels"`
	ForceDisabled                     types.Bool   `tfsdk:"force_disabled"`
	IsJumpServer                      types.Bool   `tfsdk:"is_jump_server"`
	IsSwarmManager                    types.Bool   `tfsdk:"is_swarm_manager"`
	IsSwarmWorker                     types.Bool   `tfsdk:"is_swarm_worker"`
	IsSentinelDebugEnabled            types.Bool   `tfsdk:"is_sentinel_debug_enabled"`
	DisableApplicationImageRetention  types.Bool   `tfsdk:"disable_application_image_retention"`
	SentinelToken                     types.String `tfsdk:"sentinel_token"`
	SentinelCustomURL                 types.String `tfsdk:"sentinel_custom_url"`
	IsLogdrainAxiomEnabled            types.Bool   `tfsdk:"is_logdrain_axiom_enabled"`
	LogdrainAxiomAPIKey               types.String `tfsdk:"logdrain_axiom_api_key"`
	LogdrainAxiomDatasetName          types.String `tfsdk:"logdrain_axiom_dataset_name"`
	IsLogdrainCustomEnabled           types.Bool   `tfsdk:"is_logdrain_custom_enabled"`
	LogdrainCustomConfig              types.String `tfsdk:"logdrain_custom_config"`
	LogdrainCustomConfigParser        types.String `tfsdk:"logdrain_custom_config_parser"`
	IsLogdrainHighlightEnabled        types.Bool   `tfsdk:"is_logdrain_highlight_enabled"`
	LogdrainHighlightProjectID        types.String `tfsdk:"logdrain_highlight_project_id"`
	IsLogdrainNewrelicEnabled         types.Bool   `tfsdk:"is_logdrain_newrelic_enabled"`
	LogdrainNewrelicBaseURI           types.String `tfsdk:"logdrain_newrelic_base_uri"`
	LogdrainNewrelicLicenseKey        types.String `tfsdk:"logdrain_newrelic_license_key"`
	ComposeVersion                    types.String `tfsdk:"compose_version"`
	ComposeVersionCheckedAt           types.String `tfsdk:"compose_version_checked_at"`
	DockerVersion                     types.String `tfsdk:"docker_version"`
	DockerVersionCheckedAt            types.String `tfsdk:"docker_version_checked_at"`
	BackupCompressionCPUPercentage    types.Int64  `tfsdk:"backup_compression_cpu_percentage"`
}

// NewResource returns a new server resource.
func NewResource() resource.Resource {
	return &serverResource{}
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify server.\n\n~> **Warning:** Deleting a server will cascade-delete all applications, databases, and services deployed on it.",
		Attributes: CommonServerAttrs(ctx, map[string]schema.Attribute{
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the server.",
				Required:            true,
			},
		}),
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_server"})

	input := client.CreateServerInput{
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		IP:             plan.IP.ValueString(),
		Port:           int(plan.Port.ValueInt64()),
		User:           plan.User.ValueString(),
		PrivateKeyUUID: plan.PrivateKeyUUID.ValueString(),
		IsBuildServer:  flex.BoolValueOrNull(plan.IsBuildServer),
	}

	created, err := r.client.CreateServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating server",
			fmt.Sprintf("server %q (IP %s): %s", plan.Name.ValueString(), plan.IP.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.IsReachable.IsUnknown() {
		plan.IsReachable = types.BoolNull()
	}
	if plan.IsUsable.IsUnknown() {
		plan.IsUsable = types.BoolNull()
	}
	if plan.ServerDiskUsageCheckFrequency.IsUnknown() {
		plan.ServerDiskUsageCheckFrequency = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Create endpoint only accepts core fields (name, ip, port, etc.).
	// Settings like concurrent_builds and dynamic_timeout must be sent via
	// a follow-up PATCH if the user configured non-default values.
	if hasNonDefaultSettings(plan) {
		settingsUpdate := BuildPostCreateSettingsInput(plan.commonPtrs())
		if _, err := r.client.UpdateServer(ctx, created.UUID, settingsUpdate); err != nil {
			resp.Diagnostics.AddError("Error setting server settings",
				fmt.Sprintf("server %s: %s", created.UUID, err))
			return
		}
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			flex.CreateReadBackFailedSummary("Server"),
			flex.CreateReadBackFailedDetail("server", created.UUID, err),
		)
		return
	}

	flattenServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": created.UUID})
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

	input := BuildServerUpdateInput(plan.commonPtrs(), state.commonPtrs())

	_, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_server", "uuid": state.UUID.ValueString()})
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// hasNonDefaultSettings delegates to the shared HasNonDefaultSettings helper.
func hasNonDefaultSettings(plan serverResourceModel) bool {
	return HasNonDefaultSettings(plan.commonPtrs())
}

func (m *serverResourceModel) commonPtrs() ServerCommonPtrs {
	return ServerCommonPtrs{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		IP: &m.IP, User: &m.User, PrivateKeyUUID: &m.PrivateKeyUUID,
		Port: &m.Port, ConcurrentBuilds: &m.ConcurrentBuilds, DynamicTimeout: &m.DynamicTimeout,
		DeploymentQueueLimit:                 &m.DeploymentQueueLimit,
		ConnectionTimeout:                    &m.ConnectionTimeout,
		ServerDiskUsageNotificationThreshold: &m.ServerDiskUsageNotificationThreshold,
		ServerDiskUsageCheckFrequency:        &m.ServerDiskUsageCheckFrequency,
		IsBuildServer:                        &m.IsBuildServer, IsReachable: &m.IsReachable, IsUsable: &m.IsUsable,
		WildcardDomain: &m.WildcardDomain, IsCloudFlareTunnel: &m.IsCloudFlareTunnel,
		ServerTimezone: &m.ServerTimezone, IsMetricsEnabled: &m.IsMetricsEnabled,
		IsTerminalEnabled: &m.IsTerminalEnabled, IsSentinelEnabled: &m.IsSentinelEnabled,
		SentinelMetricsHistoryDays: &m.SentinelMetricsHistoryDays, SentinelMetricsRefreshRateSeconds: &m.SentinelMetricsRefreshRateSeconds,
		SentinelPushIntervalSeconds: &m.SentinelPushIntervalSeconds,
		DockerCleanupFrequency:      &m.DockerCleanupFrequency, DockerCleanupThreshold: &m.DockerCleanupThreshold,
		ForceDockerCleanup: &m.ForceDockerCleanup, DeleteUnusedVolumes: &m.DeleteUnusedVolumes,
		DeleteUnusedNetworks: &m.DeleteUnusedNetworks, GenerateExactLabels: &m.GenerateExactLabels,
		ForceDisabled: &m.ForceDisabled, IsJumpServer: &m.IsJumpServer,
		IsSwarmManager: &m.IsSwarmManager, IsSwarmWorker: &m.IsSwarmWorker,
		IsSentinelDebugEnabled: &m.IsSentinelDebugEnabled, DisableApplicationImageRetention: &m.DisableApplicationImageRetention,
		SentinelToken: &m.SentinelToken, SentinelCustomURL: &m.SentinelCustomURL,
		IsLogdrainAxiomEnabled: &m.IsLogdrainAxiomEnabled, LogdrainAxiomAPIKey: &m.LogdrainAxiomAPIKey,
		LogdrainAxiomDatasetName: &m.LogdrainAxiomDatasetName, IsLogdrainCustomEnabled: &m.IsLogdrainCustomEnabled,
		LogdrainCustomConfig: &m.LogdrainCustomConfig, LogdrainCustomConfigParser: &m.LogdrainCustomConfigParser,
		IsLogdrainHighlightEnabled: &m.IsLogdrainHighlightEnabled, LogdrainHighlightProjectID: &m.LogdrainHighlightProjectID,
		IsLogdrainNewrelicEnabled: &m.IsLogdrainNewrelicEnabled, LogdrainNewrelicBaseURI: &m.LogdrainNewrelicBaseURI,
		LogdrainNewrelicLicenseKey: &m.LogdrainNewrelicLicenseKey,
		ComposeVersion:             &m.ComposeVersion, ComposeVersionCheckedAt: &m.ComposeVersionCheckedAt,
		DockerVersion: &m.DockerVersion, DockerVersionCheckedAt: &m.DockerVersionCheckedAt,
		BackupCompressionCPUPercentage: &m.BackupCompressionCPUPercentage,
	}
}

func flattenServer(srv *client.Server, model *serverResourceModel) {
	FlattenServerCommon(srv, model.commonPtrs())
}
