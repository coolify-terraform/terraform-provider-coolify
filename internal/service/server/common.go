package server

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServerCommonPtrs holds pointers to the model fields shared by all server
// resource types (coolify_server, coolify_server_hetzner).
type ServerCommonPtrs struct {
	UUID, Name, Description, IP, User, PrivateKeyUUID *types.String
	Port, ConcurrentBuilds, DynamicTimeout            *types.Int64
	DeploymentQueueLimit, ConnectionTimeout           *types.Int64
	ServerDiskUsageNotificationThreshold              *types.Int64
	ServerDiskUsageCheckFrequency                     *types.String
	IsBuildServer, IsReachable, IsUsable              *types.Bool
	// Read-only extended settings returned by GET responses.
	WildcardDomain                    *types.String
	IsCloudFlareTunnel                *types.Bool
	ServerTimezone                    *types.String
	IsMetricsEnabled                  *types.Bool
	IsTerminalEnabled                 *types.Bool
	IsSentinelEnabled                 *types.Bool
	SentinelMetricsHistoryDays        *types.Int64
	SentinelMetricsRefreshRateSeconds *types.Int64
	SentinelPushIntervalSeconds       *types.Int64
	DockerCleanupFrequency            *types.String
	DockerCleanupThreshold            *types.Int64
	ForceDockerCleanup                *types.Bool
	DeleteUnusedVolumes               *types.Bool
	DeleteUnusedNetworks              *types.Bool
	GenerateExactLabels               *types.Bool
	ForceDisabled                     *types.Bool
	IsJumpServer                      *types.Bool
	IsSwarmManager                    *types.Bool
	IsSwarmWorker                     *types.Bool
	IsSentinelDebugEnabled            *types.Bool
	DisableApplicationImageRetention  *types.Bool
	SentinelToken                     *types.String
	SentinelCustomURL                 *types.String
	IsLogdrainAxiomEnabled            *types.Bool
	LogdrainAxiomAPIKey               *types.String
	LogdrainAxiomDatasetName          *types.String
	IsLogdrainCustomEnabled           *types.Bool
	LogdrainCustomConfig              *types.String
	LogdrainCustomConfigParser        *types.String
	IsLogdrainHighlightEnabled        *types.Bool
	LogdrainHighlightProjectID        *types.String
	IsLogdrainNewrelicEnabled         *types.Bool
	LogdrainNewrelicBaseURI           *types.String
	LogdrainNewrelicLicenseKey        *types.String
	ComposeVersion                    *types.String
	ComposeVersionCheckedAt           *types.String
	DockerVersion                     *types.String
	DockerVersionCheckedAt            *types.String
}

// CommonServerAttrs returns the schema attributes shared by all server
// resource types. Callers pass type-specific attributes via extra which
// are merged into the returned map.
func CommonServerAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the server.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the server.",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the server.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"port": schema.Int64Attribute{
			MarkdownDescription: "The SSH port of the server.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(22),
			Validators:          []validator.Int64{int64validator.Between(1, 65535)},
		},
		"user": schema.StringAttribute{
			MarkdownDescription: "The SSH user for connecting to the server.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("root"),
		},
		"private_key_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the private key used for SSH authentication.",
			Required:            true,
			Validators:          []validator.String{validate.UUID()},
		},
		"is_build_server": schema.BoolAttribute{
			MarkdownDescription: "Whether this server is used for building applications.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_reachable": schema.BoolAttribute{
			MarkdownDescription: "Whether the server is currently reachable.",
			Computed:            true,
		},
		"is_usable": schema.BoolAttribute{
			MarkdownDescription: "Whether the server is currently usable for deployments.",
			Computed:            true,
		},
		"concurrent_builds": schema.Int64Attribute{
			MarkdownDescription: "How many deployments can run in parallel on this server.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(2),
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"dynamic_timeout": schema.Int64Attribute{
			MarkdownDescription: "Timeout in seconds for Docker operations (pull, build, health check) during deployment.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(3600),
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"deployment_queue_limit": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of queued deployments (default 25).",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(25),
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"connection_timeout": schema.Int64Attribute{
			MarkdownDescription: "SSH connection timeout in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(10),
			Validators:          []validator.Int64{int64validator.Between(1, 300)},
		},
		"server_disk_usage_notification_threshold": schema.Int64Attribute{
			MarkdownDescription: "Disk usage percentage at which a notification is sent.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(80),
			Validators:          []validator.Int64{int64validator.Between(1, 100)},
		},
		"server_disk_usage_check_frequency": schema.StringAttribute{
			MarkdownDescription: "Cron or Coolify human schedule for how often disk usage is checked (e.g., `*/5 * * * *`, `daily`, `@daily`).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators:          []validator.String{validate.CoolifyFrequency()},
		},
	}
	addExtendedSettingsAttrs(attrs)
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// addExtendedSettingsAttrs adds read-only extended server settings returned by the API.
func addExtendedSettingsAttrs(attrs map[string]schema.Attribute) {
	addCoreExtendedSettingsAttrs(attrs)
	addLogdrainSettingsAttrs(attrs)
	addHostProbeVersionAttrs(attrs)
}

func addCoreExtendedSettingsAttrs(attrs map[string]schema.Attribute) {
	attrs["wildcard_domain"] = schema.StringAttribute{
		MarkdownDescription: "Wildcard domain for applications on this server (e.g., `example.com`).",
		Computed:            true,
	}
	attrs["is_cloudflare_tunnel"] = schema.BoolAttribute{
		MarkdownDescription: "Whether this server uses a Cloudflare Tunnel.",
		Computed:            true,
	}
	attrs["server_timezone"] = schema.StringAttribute{
		MarkdownDescription: "Server timezone (e.g., `UTC`, `America/New_York`).",
		Computed:            true,
	}
	attrs["is_metrics_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether metrics collection is enabled on this server.",
		Computed:            true,
	}
	attrs["is_terminal_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether the web terminal is enabled for this server.",
		Computed:            true,
	}
	attrs["is_sentinel_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether the Sentinel monitoring agent is enabled.",
		Computed:            true,
	}
	attrs["sentinel_metrics_history_days"] = schema.Int64Attribute{
		MarkdownDescription: "Number of days to retain Sentinel metrics.",
		Computed:            true,
	}
	attrs["sentinel_metrics_refresh_rate_seconds"] = schema.Int64Attribute{
		MarkdownDescription: "Sentinel metrics refresh rate in seconds.",
		Computed:            true,
	}
	attrs["sentinel_push_interval_seconds"] = schema.Int64Attribute{
		MarkdownDescription: "Interval in seconds between Sentinel metric pushes.",
		Computed:            true,
	}
	attrs["docker_cleanup_frequency"] = schema.StringAttribute{
		MarkdownDescription: "Cron expression for Docker cleanup schedule.",
		Computed:            true,
	}
	attrs["docker_cleanup_threshold"] = schema.Int64Attribute{
		MarkdownDescription: "Disk usage percentage threshold for Docker cleanup.",
		Computed:            true,
	}
	attrs["force_docker_cleanup"] = schema.BoolAttribute{
		MarkdownDescription: "Whether to force Docker cleanup regardless of disk usage.",
		Computed:            true,
	}
	attrs["delete_unused_volumes"] = schema.BoolAttribute{
		MarkdownDescription: "Whether to delete unused Docker volumes during cleanup.",
		Computed:            true,
	}
	attrs["delete_unused_networks"] = schema.BoolAttribute{
		MarkdownDescription: "Whether to delete unused Docker networks during cleanup.",
		Computed:            true,
	}
	attrs["generate_exact_labels"] = schema.BoolAttribute{
		MarkdownDescription: "Whether to generate exact Docker labels (removes extra labels from containers).",
		Computed:            true,
	}
	attrs["force_disabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether the server is force-disabled in Coolify. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["is_jump_server"] = schema.BoolAttribute{
		MarkdownDescription: "Whether this server is a jump host. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["is_swarm_manager"] = schema.BoolAttribute{
		MarkdownDescription: "Whether this server is a Docker Swarm manager. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["is_swarm_worker"] = schema.BoolAttribute{
		MarkdownDescription: "Whether this server is a Docker Swarm worker. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["is_sentinel_debug_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether Sentinel debug mode is enabled. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["disable_application_image_retention"] = schema.BoolAttribute{
		MarkdownDescription: "Whether application image retention is disabled. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["sentinel_token"] = schema.StringAttribute{
		MarkdownDescription: "Sentinel agent token. Sensitive; only returned when the API token can read secrets. Read-only.",
		Computed:            true,
		Sensitive:           true,
	}
	attrs["sentinel_custom_url"] = schema.StringAttribute{
		MarkdownDescription: "Custom Sentinel push URL. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
}

func addLogdrainSettingsAttrs(attrs map[string]schema.Attribute) {
	attrs["is_logdrain_axiom_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether Axiom log drain is enabled. Read-only (not on public server PATCH allow-list).",
		Computed:            true,
	}
	attrs["logdrain_axiom_api_key"] = schema.StringAttribute{
		MarkdownDescription: "Axiom API key for log drain. Sensitive; read-only.",
		Computed:            true,
		Sensitive:           true,
	}
	attrs["logdrain_axiom_dataset_name"] = schema.StringAttribute{
		MarkdownDescription: "Axiom dataset name for log drain. Read-only.",
		Computed:            true,
	}
	attrs["is_logdrain_custom_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether custom log drain is enabled. Read-only.",
		Computed:            true,
	}
	attrs["logdrain_custom_config"] = schema.StringAttribute{
		MarkdownDescription: "Custom log drain configuration. Sensitive; read-only.",
		Computed:            true,
		Sensitive:           true,
	}
	attrs["logdrain_custom_config_parser"] = schema.StringAttribute{
		MarkdownDescription: "Custom log drain config parser. Read-only.",
		Computed:            true,
	}
	attrs["is_logdrain_highlight_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether Highlight log drain is enabled. Read-only.",
		Computed:            true,
	}
	attrs["logdrain_highlight_project_id"] = schema.StringAttribute{
		MarkdownDescription: "Highlight project ID for log drain. Read-only.",
		Computed:            true,
	}
	attrs["is_logdrain_newrelic_enabled"] = schema.BoolAttribute{
		MarkdownDescription: "Whether New Relic log drain is enabled. Read-only.",
		Computed:            true,
	}
	attrs["logdrain_newrelic_base_uri"] = schema.StringAttribute{
		MarkdownDescription: "New Relic base URI for log drain. Read-only.",
		Computed:            true,
	}
	attrs["logdrain_newrelic_license_key"] = schema.StringAttribute{
		MarkdownDescription: "New Relic license key for log drain. Sensitive; read-only.",
		Computed:            true,
		Sensitive:           true,
	}
}

// addHostProbeVersionAttrs adds Coolify tip/main host probe fields (expected >= 4.3.2).
func addHostProbeVersionAttrs(attrs map[string]schema.Attribute) {
	attrs["compose_version"] = schema.StringAttribute{
		MarkdownDescription: "Docker Compose version reported by Coolify for this server (host probe). " +
			"Read-only; Populated on Coolify >= v4.3.2 when the host has been probed. Empty on older instances.",
		Computed: true,
	}
	attrs["compose_version_checked_at"] = schema.StringAttribute{
		MarkdownDescription: "When Coolify last checked the Compose version on this server. Read-only.",
		Computed:            true,
	}
	attrs["docker_version"] = schema.StringAttribute{
		MarkdownDescription: "Docker engine version reported by Coolify for this server (host probe). " +
			"Read-only; Populated on Coolify >= v4.3.2 when the host has been probed. Empty on older instances.",
		Computed: true,
	}
	attrs["docker_version_checked_at"] = schema.StringAttribute{
		MarkdownDescription: "When Coolify last checked the Docker version on this server. Read-only.",
		Computed:            true,
	}
}

// FlattenServerCommon sets the fields shared by all server resource types
// from the API response.
func FlattenServerCommon(srv *client.Server, f ServerCommonPtrs) {
	*f.UUID = types.StringValue(srv.UUID)
	*f.Name = types.StringValue(srv.Name)
	*f.Description = flex.StringToFramework(srv.Description)
	*f.IP = types.StringValue(srv.IP)
	*f.Port = types.Int64Value(int64(srv.Port))
	*f.User = types.StringValue(srv.User)
	// Only overwrite private_key_uuid when the API returns a non-empty
	// value. The GET response omits this field, which would cause
	// "inconsistent result after apply" if we set it to "".
	if srv.PrivateKeyUUID != "" {
		*f.PrivateKeyUUID = types.StringValue(srv.PrivateKeyUUID)
	}
	*f.IsBuildServer = types.BoolValue(srv.IsBuildServer)
	*f.IsReachable = types.BoolValue(srv.IsReachable)
	*f.IsUsable = types.BoolValue(srv.IsUsable)

	if srv.Settings != nil {
		connectionTimeout := srv.Settings.ConnectionTimeout
		if connectionTimeout == 0 {
			connectionTimeout = 10
		}
		*f.ConcurrentBuilds = types.Int64Value(int64(srv.Settings.ConcurrentBuilds))
		*f.DynamicTimeout = types.Int64Value(int64(srv.Settings.DynamicTimeout))
		*f.DeploymentQueueLimit = types.Int64Value(int64(srv.Settings.DeploymentQueueLimit))
		*f.ConnectionTimeout = types.Int64Value(int64(connectionTimeout))
		*f.ServerDiskUsageNotificationThreshold = types.Int64Value(int64(srv.Settings.ServerDiskUsageNotificationThreshold))
		*f.ServerDiskUsageCheckFrequency = flex.StringToFramework(srv.Settings.ServerDiskUsageCheckFrequency)
		flattenExtendedSettings(srv.Settings, f)
	}
}

// flattenExtendedSettings sets the extended settings fields from the API response.
func flattenExtendedSettings(s *client.ServerSettings, f ServerCommonPtrs) {
	*f.WildcardDomain = flex.StringToFramework(s.WildcardDomain)
	*f.IsCloudFlareTunnel = types.BoolValue(s.IsCloudFlareTunnel)
	*f.ServerTimezone = flex.StringToFramework(s.ServerTimezone)
	*f.IsMetricsEnabled = types.BoolValue(s.IsMetricsEnabled)
	*f.IsTerminalEnabled = types.BoolValue(s.IsTerminalEnabled)
	*f.IsSentinelEnabled = types.BoolValue(s.IsSentinelEnabled)
	*f.SentinelMetricsHistoryDays = types.Int64Value(int64(s.SentinelMetricsHistoryDays))
	*f.SentinelMetricsRefreshRateSeconds = types.Int64Value(int64(s.SentinelMetricsRefreshRateSeconds))
	*f.SentinelPushIntervalSeconds = types.Int64Value(int64(s.SentinelPushIntervalSeconds))
	*f.DockerCleanupFrequency = flex.StringToFramework(s.DockerCleanupFrequency)
	*f.DockerCleanupThreshold = types.Int64Value(int64(s.DockerCleanupThreshold))
	*f.ForceDockerCleanup = types.BoolValue(s.ForceDockerCleanup)
	*f.DeleteUnusedVolumes = types.BoolValue(s.DeleteUnusedVolumes)
	*f.DeleteUnusedNetworks = types.BoolValue(s.DeleteUnusedNetworks)
	*f.GenerateExactLabels = types.BoolValue(s.GenerateExactLabels)
	setBoolPtr(f.ForceDisabled, s.IsForceDisabled)
	setBoolPtr(f.IsJumpServer, s.IsJumpServer)
	setBoolPtr(f.IsSwarmManager, s.IsSwarmManager)
	setBoolPtr(f.IsSwarmWorker, s.IsSwarmWorker)
	setBoolPtr(f.IsSentinelDebugEnabled, s.IsSentinelDebugEnabled)
	setBoolPtr(f.DisableApplicationImageRetention, s.DisableApplicationImageRetention)
	setStringPtr(f.SentinelToken, s.SentinelToken)
	setStringPtr(f.SentinelCustomURL, s.SentinelCustomURL)
	setBoolPtr(f.IsLogdrainAxiomEnabled, s.IsLogdrainAxiomEnabled)
	setStringPtr(f.LogdrainAxiomAPIKey, s.LogdrainAxiomAPIKey)
	setStringPtr(f.LogdrainAxiomDatasetName, s.LogdrainAxiomDatasetName)
	setBoolPtr(f.IsLogdrainCustomEnabled, s.IsLogdrainCustomEnabled)
	setStringPtr(f.LogdrainCustomConfig, s.LogdrainCustomConfig)
	setStringPtr(f.LogdrainCustomConfigParser, s.LogdrainCustomConfigParser)
	setBoolPtr(f.IsLogdrainHighlightEnabled, s.IsLogdrainHighlightEnabled)
	setStringPtr(f.LogdrainHighlightProjectID, s.LogdrainHighlightProjectID)
	setBoolPtr(f.IsLogdrainNewrelicEnabled, s.IsLogdrainNewrelicEnabled)
	setStringPtr(f.LogdrainNewrelicBaseURI, s.LogdrainNewrelicBaseURI)
	setStringPtr(f.LogdrainNewrelicLicenseKey, s.LogdrainNewrelicLicenseKey)
	setStringPtr(f.ComposeVersion, s.ComposeVersion)
	setStringPtr(f.ComposeVersionCheckedAt, s.ComposeVersionCheckedAt)
	setStringPtr(f.DockerVersion, s.DockerVersion)
	setStringPtr(f.DockerVersionCheckedAt, s.DockerVersionCheckedAt)
}

func setBoolPtr(dst *types.Bool, v bool) {
	if dst == nil {
		return
	}
	*dst = types.BoolValue(v)
}

func setStringPtr(dst *types.String, v string) {
	if dst == nil {
		return
	}
	*dst = flex.StringToFramework(v)
}

// HasNonDefaultSettings returns true if any settings field in the plan
// differs from Coolify's create-time default. Used by Create methods to
// decide whether a post-create PATCH is needed for settings.
func HasNonDefaultSettings(p ServerCommonPtrs) bool {
	return flex.Int64ValueNonDefault(*p.ConcurrentBuilds, 2) ||
		flex.Int64ValueNonDefault(*p.DynamicTimeout, 3600) ||
		flex.Int64ValueNonDefault(*p.DeploymentQueueLimit, 25) ||
		flex.Int64ValueNonDefault(*p.ConnectionTimeout, 10) ||
		flex.Int64ValueNonDefault(*p.ServerDiskUsageNotificationThreshold, 80) ||
		flex.StringValueNonDefault(*p.ServerDiskUsageCheckFrequency, "")
}

// BuildPostCreateSettingsInput returns an UpdateServerInput populated with
// only the settings fields that differ from Coolify's create-time defaults.
// Callers can extend the returned input with additional fields before sending.
func BuildPostCreateSettingsInput(p ServerCommonPtrs) client.UpdateServerInput {
	return client.UpdateServerInput{
		ConcurrentBuilds:                     flex.IntIfNonDefault(*p.ConcurrentBuilds, 2),
		DynamicTimeout:                       flex.IntIfNonDefault(*p.DynamicTimeout, 3600),
		DeploymentQueueLimit:                 flex.IntIfNonDefault(*p.DeploymentQueueLimit, 25),
		ConnectionTimeout:                    flex.IntIfNonDefault(*p.ConnectionTimeout, 10),
		ServerDiskUsageNotificationThreshold: flex.IntIfNonDefault(*p.ServerDiskUsageNotificationThreshold, 80),
		ServerDiskUsageCheckFrequency:        flex.StringValueOrNull(*p.ServerDiskUsageCheckFrequency),
	}
}

// HasNonDefaultCloudProviderSettings reports whether a cloud-provisioned
// server (Hetzner/DigitalOcean/Vultr) needs a post-create PATCH. Cloud create
// endpoints only accept provider-specific fields, so description/port/user/
// is_build_server and extended settings must be applied afterward when non-default.
func HasNonDefaultCloudProviderSettings(p ServerCommonPtrs) bool {
	return flex.StringValueNonDefault(*p.Description, "") ||
		flex.Int64ValueNonDefault(*p.Port, 22) ||
		flex.StringValueNonDefault(*p.User, "root") ||
		flex.BoolValueNonDefault(*p.IsBuildServer, false) ||
		HasNonDefaultSettings(p)
}

// BuildPostCreateCloudProviderInput builds the follow-up PATCH body for
// Hetzner/DigitalOcean/Vultr servers after provider-specific create.
func BuildPostCreateCloudProviderInput(p ServerCommonPtrs) client.UpdateServerInput {
	input := BuildPostCreateSettingsInput(p)
	// Core fields the cloud create endpoints do not accept.
	input.Description = flex.StringValueOrNull(*p.Description)
	input.Port = flex.IntIfNonDefault(*p.Port, 22)
	input.User = flex.StringValueOrNull(*p.User)
	input.IsBuildServer = flex.BoolValueOrNull(*p.IsBuildServer)
	return input
}

// NormalizeUnknownCloudServerPlanFields resolves Optional+Computed fields
// that remain unknown after create so partial state is valid before
// read-back (cloud providers often leave IP/reachability unknown at create).
func NormalizeUnknownCloudServerPlanFields(p ServerCommonPtrs) {
	if p.Description != nil && p.Description.IsUnknown() {
		*p.Description = types.StringNull()
	}
	if p.IP != nil && p.IP.IsUnknown() {
		*p.IP = types.StringNull()
	}
	if p.IsReachable != nil && p.IsReachable.IsUnknown() {
		*p.IsReachable = types.BoolNull()
	}
	if p.IsUsable != nil && p.IsUsable.IsUnknown() {
		*p.IsUsable = types.BoolNull()
	}
	if p.ServerDiskUsageCheckFrequency != nil && p.ServerDiskUsageCheckFrequency.IsUnknown() {
		*p.ServerDiskUsageCheckFrequency = types.StringNull()
	}
}

// ApplyPostCreateCloudProviderSettings sends the shared post-create PATCH
// when HasNonDefaultCloudProviderSettings is true. No-op when all defaults.
func ApplyPostCreateCloudProviderSettings(ctx context.Context, c *client.Client, uuid string, p ServerCommonPtrs) error {
	if !HasNonDefaultCloudProviderSettings(p) {
		return nil
	}
	if _, err := c.UpdateServer(ctx, uuid, BuildPostCreateCloudProviderInput(p)); err != nil {
		return fmt.Errorf("server %s: %w", uuid, err)
	}
	return nil
}

// BuildServerUpdateInput constructs an UpdateServerInput from the diff
// between plan and state for the shared server fields.
func BuildServerUpdateInput(plan, state ServerCommonPtrs) client.UpdateServerInput {
	input := client.UpdateServerInput{
		Name:                                 flex.StringIfChanged(*plan.Name, *state.Name),
		Description:                          flex.StringIfChanged(*plan.Description, *state.Description),
		IP:                                   flex.StringIfChanged(*plan.IP, *state.IP),
		Port:                                 flex.IntIfChanged(*plan.Port, *state.Port),
		User:                                 flex.StringIfChanged(*plan.User, *state.User),
		PrivateKeyUUID:                       flex.StringIfChanged(*plan.PrivateKeyUUID, *state.PrivateKeyUUID),
		IsBuildServer:                        flex.BoolIfChanged(*plan.IsBuildServer, *state.IsBuildServer),
		ConcurrentBuilds:                     flex.IntIfChanged(*plan.ConcurrentBuilds, *state.ConcurrentBuilds),
		DynamicTimeout:                       flex.IntIfChanged(*plan.DynamicTimeout, *state.DynamicTimeout),
		DeploymentQueueLimit:                 flex.IntIfChanged(*plan.DeploymentQueueLimit, *state.DeploymentQueueLimit),
		ConnectionTimeout:                    flex.IntIfChanged(*plan.ConnectionTimeout, *state.ConnectionTimeout),
		ServerDiskUsageNotificationThreshold: flex.IntIfChanged(*plan.ServerDiskUsageNotificationThreshold, *state.ServerDiskUsageNotificationThreshold),
		ServerDiskUsageCheckFrequency:        flex.StringIfChanged(*plan.ServerDiskUsageCheckFrequency, *state.ServerDiskUsageCheckFrequency),
	}
	return input
}
