package server

import (
	"context"
	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
			MarkdownDescription: "Cron expression for how often disk usage is checked (e.g., `*/5 * * * *` or `@daily`).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators: []validator.String{
				stringvalidator.RegexMatches(
					regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
					"must be a valid cron expression (e.g., \"*/5 * * * *\" or \"@daily\")",
				),
			},
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
