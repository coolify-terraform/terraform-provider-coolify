package vultr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/server"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &vultrServerResource{}
	_ resource.ResourceWithConfigure   = &vultrServerResource{}
	_ resource.ResourceWithImportState = &vultrServerResource{}
)

type vultrServerResource struct {
	client *client.Client
}

type vultrServerResourceModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`
	// Computed after create.
	UUID        types.String `tfsdk:"uuid"`
	IP          types.String `tfsdk:"ip"`
	IsReachable types.Bool   `tfsdk:"is_reachable"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`

	// Vultr create-only fields (RequiresReplace).
	CloudProviderTokenUUID types.String `tfsdk:"cloud_provider_token_uuid"`
	Plan                   types.String `tfsdk:"plan"`
	Region                 types.String `tfsdk:"region"`
	OsID                   types.Int64  `tfsdk:"os_id"`
	VultrSSHKeyIDs         types.String `tfsdk:"vultr_ssh_key_ids"`
	CloudInitScript        types.String `tfsdk:"cloud_init_script"`
	InstantValidate        types.Bool   `tfsdk:"instant_validate"`
	EnableIPv6             types.Bool   `tfsdk:"enable_ipv6"`
	DisablePublicIPv4      types.Bool   `tfsdk:"disable_public_ipv4"`

	// Shared server fields (updatable).
	Name                                 types.String `tfsdk:"name"`
	Description                          types.String `tfsdk:"description"`
	Port                                 types.Int64  `tfsdk:"port"`
	User                                 types.String `tfsdk:"user"`
	PrivateKeyUUID                       types.String `tfsdk:"private_key_uuid"`
	IsBuildServer                        types.Bool   `tfsdk:"is_build_server"`
	ConcurrentBuilds                     types.Int64  `tfsdk:"concurrent_builds"`
	DynamicTimeout                       types.Int64  `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                 types.Int64  `tfsdk:"deployment_queue_limit"`
	ConnectionTimeout                    types.Int64  `tfsdk:"connection_timeout"`
	ServerDiskUsageNotificationThreshold types.Int64  `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        types.String `tfsdk:"server_disk_usage_check_frequency"`
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

// NewResource returns a new Vultr server resource.
func NewResource() resource.Resource {
	return &vultrServerResource{}
}

func (r *vultrServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_vultr"
}

func (r *vultrServerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provisions a Vultr instance and registers it with Coolify. Requires Coolify >= v4.2.0.\n\n~> **Warning:** Deleting this resource will delete the server from Coolify and cascade-delete all applications, databases, and services deployed on it. The underlying Vultr server is not destroyed; manage its lifecycle separately.\n\n~> **Import note:** Vultr-specific fields (`cloud_provider_token_uuid`, `plan`, `region`, `os_id`, `vultr_ssh_key_ids`, `cloud_init_script`) are only sent at creation time and are not returned by the Coolify API. After `terraform import`, these fields will be empty in state. Set them in your configuration before running `terraform plan` to avoid a forced replacement.",
		Attributes:          server.CommonServerAttrs(ctx, vultrSchemaAttributes()),
	}
}

func vultrSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP address assigned to the server by Vultr.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		// Vultr create-only fields.
		"cloud_provider_token_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the Vultr cloud provider token (from `coolify_cloud_token`). Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"plan": schema.StringAttribute{
			MarkdownDescription: "The Vultr server type (e.g., `s-1vcpu-1gb`). Use `coolify_vultr_plans` to list available plans. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"region": schema.StringAttribute{
			MarkdownDescription: "The Vultr region slug (e.g., `nyc1`). Use `coolify_vultr_regions` to list available regions. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"os_id": schema.Int64Attribute{
			MarkdownDescription: "The Vultr operating system ID. Use `coolify_vultr_os` to list available operating systems. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
		},
		"vultr_ssh_key_ids": schema.StringAttribute{
			MarkdownDescription: "Comma-separated list of Vultr SSH key IDs (strings). Use `coolify_vultr_ssh_keys` to list available keys. Changing this forces a new resource.",
			Optional:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"cloud_init_script": schema.StringAttribute{
			MarkdownDescription: "Cloud-init script to run on server creation. Changing this forces a new resource.",
			Optional:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"instant_validate": schema.BoolAttribute{
			MarkdownDescription: "Whether to validate server connectivity immediately after creation. Defaults to false to match Coolify.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"enable_ipv6": schema.BoolAttribute{
			MarkdownDescription: "Whether to enable IPv6 on the Vultr instance. Defaults to true to match Coolify.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"disable_public_ipv4": schema.BoolAttribute{
			MarkdownDescription: "Whether to disable public IPv4 on the Vultr instance.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

func (r *vultrServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *vultrServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vultrServerResourceModel
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_server_vultr"})

	input := client.CreateVultrServerInput{
		Name:                   plan.Name.ValueString(),
		CloudProviderTokenUUID: plan.CloudProviderTokenUUID.ValueString(),
		Region:                 plan.Region.ValueString(),
		Plan:                   plan.Plan.ValueString(),
		OsID:                   plan.OsID.ValueInt64(),
		PrivateKeyUUID:         plan.PrivateKeyUUID.ValueString(),
		EnableIPv6:             flex.BoolValueOrNull(plan.EnableIPv6),
		DisablePublicIPv4:      flex.BoolValueOrNull(plan.DisablePublicIPv4),
		InstantValidate:        flex.BoolValueOrNull(plan.InstantValidate),
	}
	if !plan.VultrSSHKeyIDs.IsNull() && !plan.VultrSSHKeyIDs.IsUnknown() && plan.VultrSSHKeyIDs.ValueString() != "" {
		for _, part := range strings.Split(plan.VultrSSHKeyIDs.ValueString(), ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				input.VultrSSHKeyIDs = append(input.VultrSSHKeyIDs, part)
			}
		}
	}
	flex.SetIfKnown(&input.CloudInitScript, plan.CloudInitScript)

	created, err := r.client.CreateVultrServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Vultr server",
			fmt.Sprintf("server %q: %s", plan.Name.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	server.NormalizeUnknownCloudServerPlanFields(plan.commonPtrs())

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cloud create endpoints only accept provider-specific fields; shared
	// settings/core fields need a follow-up PATCH when non-default.
	if err := server.ApplyPostCreateCloudProviderSettings(ctx, r.client, created.UUID, plan.commonPtrs()); err != nil {
		resp.Diagnostics.AddError("Error setting Vultr server settings", err.Error())
		return
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Vultr server created but refresh failed",
			fmt.Sprintf("Coolify created Vultr server %s, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, err),
		)
		return
	}

	flattenVultrServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": created.UUID})
}

func (r *vultrServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vultrServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Vultr server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenVultrServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *vultrServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vultrServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state vultrServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})

	input := server.BuildServerUpdateInput(plan.commonPtrs(), state.commonPtrs())

	if _, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Vultr server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Vultr server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenVultrServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "updated resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})
}

func (r *vultrServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vultrServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting Vultr server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_server_vultr", "uuid": state.UUID.ValueString()})
}

func (r *vultrServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func (m *vultrServerResourceModel) commonPtrs() server.ServerCommonPtrs {
	return server.ServerCommonPtrs{
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

func flattenVultrServer(srv *client.Server, model *vultrServerResourceModel) {
	server.FlattenServerCommon(srv, model.commonPtrs())
}
