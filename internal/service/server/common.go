package server

import (
	"context"
	"regexp"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServerCommonPtrs holds pointers to the model fields shared by all server
// resource types (coolify_server, coolify_hetzner_server).
type ServerCommonPtrs struct {
	UUID, Name, Description, IP, User, PrivateKeyUUID *types.String
	Port, ConcurrentBuilds, DynamicTimeout            *types.Int64
	DeploymentQueueLimit                              *types.Int64
	ServerDiskUsageNotificationThreshold              *types.Int64
	ServerDiskUsageCheckFrequency                     *types.String
	IsBuildServer, IsReachable, IsUsable              *types.Bool
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
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
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
		*f.ConcurrentBuilds = types.Int64Value(int64(srv.Settings.ConcurrentBuilds))
		*f.DynamicTimeout = types.Int64Value(int64(srv.Settings.DynamicTimeout))
		*f.DeploymentQueueLimit = types.Int64Value(int64(srv.Settings.DeploymentQueueLimit))
		*f.ServerDiskUsageNotificationThreshold = types.Int64Value(int64(srv.Settings.ServerDiskUsageNotificationThreshold))
		*f.ServerDiskUsageCheckFrequency = flex.StringToFramework(srv.Settings.ServerDiskUsageCheckFrequency)
	}
}

// BuildServerUpdateInput constructs an UpdateServerInput from the diff
// between plan and state for the shared server fields.
func BuildServerUpdateInput(plan, state ServerCommonPtrs) client.UpdateServerInput {
	return client.UpdateServerInput{
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
		ServerDiskUsageNotificationThreshold: flex.IntIfChanged(*plan.ServerDiskUsageNotificationThreshold, *state.ServerDiskUsageNotificationThreshold),
		ServerDiskUsageCheckFrequency:        flex.StringIfChanged(*plan.ServerDiskUsageCheckFrequency, *state.ServerDiskUsageCheckFrequency),
	}
}
