package hetzner

import (
	"context"
	"fmt"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &hetznerServerResource{}
	_ resource.ResourceWithConfigure   = &hetznerServerResource{}
	_ resource.ResourceWithImportState = &hetznerServerResource{}
)

type hetznerServerResource struct {
	client *client.Client
}

type hetznerServerResourceModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`
	// Computed after create.
	UUID        types.String `tfsdk:"uuid"`
	IP          types.String `tfsdk:"ip"`
	IsReachable types.Bool   `tfsdk:"is_reachable"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`

	// Hetzner create-only fields (RequiresReplace).
	CloudProviderTokenUUID types.String `tfsdk:"cloud_provider_token_uuid"`
	ServerType             types.String `tfsdk:"server_type"`
	Location               types.String `tfsdk:"location"`
	Image                  types.String `tfsdk:"image"`
	HetznerSSHKeyIDs       types.String `tfsdk:"hetzner_ssh_key_ids"`
	CloudInitScript        types.String `tfsdk:"cloud_init_script"`
	InstantValidate        types.Bool   `tfsdk:"instant_validate"`
	EnableIPv4             types.Bool   `tfsdk:"enable_ipv4"`
	EnableIPv6             types.Bool   `tfsdk:"enable_ipv6"`

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
}

// NewResource returns a new Hetzner server resource.
func NewResource() resource.Resource {
	return &hetznerServerResource{}
}

func (r *hetznerServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_hetzner"
}

func (r *hetznerServerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provisions a Hetzner Cloud server and registers it with Coolify.\n\n~> **Warning:** Deleting this resource will delete the server from Coolify and cascade-delete all applications, databases, and services deployed on it. The underlying Hetzner Cloud server is not destroyed; manage its lifecycle separately.\n\n~> **Import note:** Hetzner-specific fields (`cloud_provider_token_uuid`, `server_type`, `location`, `image`, `hetzner_ssh_key_ids`, `cloud_init_script`) are only sent at creation time and are not returned by the Coolify API. After `terraform import`, these fields will be empty in state. Set them in your configuration before running `terraform plan` to avoid a forced replacement.",
		Attributes:          server.CommonServerAttrs(ctx, hetznerSchemaAttributes()),
	}
}

func hetznerSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "The IP address assigned to the server by Hetzner.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		// Hetzner create-only fields.
		"cloud_provider_token_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the Hetzner cloud provider token (from `coolify_cloud_token`). Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"server_type": schema.StringAttribute{
			MarkdownDescription: "The Hetzner server type (e.g., `cx22`, `cpx31`). Use `coolify_hetzner_server_types` data source to list available types. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"location": schema.StringAttribute{
			MarkdownDescription: "The Hetzner datacenter location (e.g., `fsn1`, `nbg1`). Use `coolify_hetzner_locations` data source to list available locations. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "The OS image to use (e.g., `ubuntu-24.04`). Use `coolify_hetzner_images` data source to list available images. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"hetzner_ssh_key_ids": schema.StringAttribute{
			MarkdownDescription: "Comma-separated list of Hetzner SSH key IDs to install on the server. Use `coolify_hetzner_ssh_keys` data source to list available keys. Changing this forces a new resource.",
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
			MarkdownDescription: "Whether to validate server connectivity immediately after creation.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"enable_ipv4": schema.BoolAttribute{
			MarkdownDescription: "Whether to enable IPv4 on the server.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"enable_ipv6": schema.BoolAttribute{
			MarkdownDescription: "Whether to enable IPv6 on the server.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
	}
}

func (r *hetznerServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *hetznerServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hetznerServerResourceModel
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_server_hetzner"})

	input := client.CreateHetznerServerInput{
		Name:                   plan.Name.ValueString(),
		CloudProviderTokenUUID: plan.CloudProviderTokenUUID.ValueString(),
		ServerType:             plan.ServerType.ValueString(),
		Location:               plan.Location.ValueString(),
		Image:                  plan.Image.ValueString(),
		PrivateKeyUUID:         plan.PrivateKeyUUID.ValueString(),
		EnableIPv4:             flex.BoolValueOrNull(plan.EnableIPv4),
		EnableIPv6:             flex.BoolValueOrNull(plan.EnableIPv6),
		InstantValidate:        flex.BoolValueOrNull(plan.InstantValidate),
	}
	flex.SetIfKnown(&input.HetznerSSHKeyIDs, plan.HetznerSSHKeyIDs)
	flex.SetIfKnown(&input.CloudInitScript, plan.CloudInitScript)

	created, err := r.client.CreateHetznerServer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Hetzner server",
			fmt.Sprintf("server %q: %s", plan.Name.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.IP.IsUnknown() {
		plan.IP = types.StringNull()
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

	// The Hetzner create endpoint only accepts Hetzner-specific fields.
	// General server fields (description, port, user, is_build_server)
	// and settings must be applied via a follow-up PATCH.
	if hasNonDefaultHetznerSettings(plan) {
		settingsUpdate := client.UpdateServerInput{
			Description:                          flex.StringValueOrNull(plan.Description),
			Port:                                 flex.IntIfNonDefault(plan.Port, 22),
			User:                                 flex.StringValueOrNull(plan.User),
			IsBuildServer:                        flex.BoolValueOrNull(plan.IsBuildServer),
			ConcurrentBuilds:                     flex.IntIfNonDefault(plan.ConcurrentBuilds, 2),
			DynamicTimeout:                       flex.IntIfNonDefault(plan.DynamicTimeout, 3600),
			DeploymentQueueLimit:                 flex.IntIfNonDefault(plan.DeploymentQueueLimit, 25),
			ConnectionTimeout:                    flex.IntIfNonDefault(plan.ConnectionTimeout, 10),
			ServerDiskUsageNotificationThreshold: flex.IntIfNonDefault(plan.ServerDiskUsageNotificationThreshold, 80),
			ServerDiskUsageCheckFrequency:        flex.StringValueOrNull(plan.ServerDiskUsageCheckFrequency),
		}
		if _, err := r.client.UpdateServer(ctx, created.UUID, settingsUpdate); err != nil {
			resp.Diagnostics.AddError("Error setting Hetzner server settings",
				fmt.Sprintf("server %s: %s", created.UUID, err))
			return
		}
	}

	// Read back for full state.
	srv, err := r.client.GetServer(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Hetzner server created but refresh failed",
			fmt.Sprintf("Coolify created Hetzner server %s, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, err),
		)
		return
	}

	flattenHetznerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": created.UUID})
}

func (r *hetznerServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenHetznerServer(srv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hetznerServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hetznerServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})

	input := server.BuildServerUpdateInput(plan.commonPtrs(), state.commonPtrs())

	if _, err := r.client.UpdateServer(ctx, state.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	srv, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Hetzner server after update", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenHetznerServer(srv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "updated resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})
}

func (r *hetznerServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hetznerServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting Hetzner server", fmt.Sprintf("server %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_server_hetzner", "uuid": state.UUID.ValueString()})
}

func (r *hetznerServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func hasNonDefaultHetznerSettings(plan hetznerServerResourceModel) bool {
	return flex.StringValueNonDefault(plan.Description, "") ||
		flex.Int64ValueNonDefault(plan.Port, 22) ||
		flex.StringValueNonDefault(plan.User, "root") ||
		flex.BoolValueNonDefault(plan.IsBuildServer, false) ||
		flex.Int64ValueNonDefault(plan.ConcurrentBuilds, 2) ||
		flex.Int64ValueNonDefault(plan.DynamicTimeout, 3600) ||
		flex.Int64ValueNonDefault(plan.DeploymentQueueLimit, 25) ||
		flex.Int64ValueNonDefault(plan.ConnectionTimeout, 10) ||
		flex.Int64ValueNonDefault(plan.ServerDiskUsageNotificationThreshold, 80) ||
		flex.StringValueNonDefault(plan.ServerDiskUsageCheckFrequency, "")
}

func (m *hetznerServerResourceModel) commonPtrs() server.ServerCommonPtrs {
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
	}
}

func flattenHetznerServer(srv *client.Server, model *hetznerServerResourceModel) {
	server.FlattenServerCommon(srv, model.commonPtrs())
}
