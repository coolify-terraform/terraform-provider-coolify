package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// commonAppFields holds pointers to the fields shared by all application
// resource models. This allows a single flatten function to write into
// any concrete model type.
type commonAppFields struct {
	UUID               *types.String
	Name               *types.String
	Description        *types.String
	GitRepository      *types.String
	GitBranch          *types.String
	BuildPack          *types.String
	PortsExposes       *types.String
	FQDN               *types.String
	DockerfileLocation *types.String
	InstallCommand     *types.String
	BuildCommand       *types.String
	StartCommand       *types.String
	Status             *types.String
	ProjectUUID        *types.String
	ServerUUID         *types.String
	EnvironmentName    *types.String
	// Resource limits
	LimitsMemory            *types.String
	LimitsMemorySwap        *types.String
	LimitsMemorySwappiness  *types.Int64
	LimitsMemoryReservation *types.String
	LimitsCPUs              *types.String
	LimitsCPUSet            *types.String
	LimitsCPUShares         *types.Int64
	// Health checks
	HealthCheckEnabled     *types.Bool
	HealthCheckPath        *types.String
	HealthCheckPort        *types.String
	HealthCheckInterval    *types.Int64
	HealthCheckTimeout     *types.Int64
	HealthCheckRetries     *types.Int64
	HealthCheckStartPeriod *types.Int64
	// Auto-deploy
	IsAutoDeployEnabled *types.Bool
}

// flattenApplicationCommon maps shared API fields into any application model
// via field pointers.
func flattenApplicationCommon(app *client.Application, f commonAppFields) {
	*f.UUID = types.StringValue(app.UUID)
	*f.Name = types.StringValue(app.Name)
	*f.Description = flex.StringToFramework(app.Description)
	// Coolify normalizes GitHub URLs by stripping the "https://github.com/"
	// prefix (e.g. "https://github.com/org/repo" becomes "org/repo"). Preserve
	// the user's original input if the API value is a suffix of it.
	if prior := f.GitRepository; !prior.IsNull() && !prior.IsUnknown() && strings.HasSuffix(prior.ValueString(), app.GitRepository) {
		*f.GitRepository = *prior
	} else {
		*f.GitRepository = types.StringValue(app.GitRepository)
	}
	*f.GitBranch = types.StringValue(app.GitBranch)
	*f.BuildPack = types.StringValue(app.BuildPack)
	// Coolify may override ports_exposes (e.g. return 80 instead of 3000
	// for Dockerfile apps). Preserve the user's configured value.
	if app.PortsExposes != "" {
		if f.PortsExposes.IsNull() || f.PortsExposes.IsUnknown() {
			*f.PortsExposes = types.StringValue(app.PortsExposes)
		}
	}
	*f.FQDN = flex.StringToFramework(app.FQDN)
	// Coolify does not return dockerfile_location on GET. Preserve from state.
	if app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	*f.InstallCommand = flex.StringToFramework(app.InstallCommand)
	*f.BuildCommand = flex.StringToFramework(app.BuildCommand)
	*f.StartCommand = flex.StringToFramework(app.StartCommand)
	*f.Status = flex.StringToFramework(app.Status)
	// Immutable fields: only update if the API returns them (Coolify may
	// omit these from the GET response).
	if app.ProjectUUID != "" {
		*f.ProjectUUID = types.StringValue(app.ProjectUUID)
	}
	if app.ServerUUID != "" {
		*f.ServerUUID = types.StringValue(app.ServerUUID)
	}
	if app.EnvironmentName != "" {
		*f.EnvironmentName = flex.StringToFramework(app.EnvironmentName)
	}
	flattenLimitsAndHealth(app, f)
}

// flattenLimitsAndHealth sets resource limits, health checks, and auto-deploy
// fields from the API response. Extracted to keep flattenApplicationCommon
// under the gocognit complexity threshold.
func flattenLimitsAndHealth(app *client.Application, f commonAppFields) {
	setStringIfNonEmpty := func(dst *types.String, v string) {
		if v != "" {
			*dst = types.StringValue(v)
		}
	}
	setStringIfNonEmpty(f.LimitsMemory, app.LimitsMemory)
	setStringIfNonEmpty(f.LimitsMemorySwap, app.LimitsMemorySwap)
	setStringIfNonEmpty(f.LimitsMemoryReservation, app.LimitsMemoryReservation)
	setStringIfNonEmpty(f.LimitsCPUs, app.LimitsCPUs)
	setStringIfNonEmpty(f.LimitsCPUSet, app.LimitsCPUSet)
	setStringIfNonEmpty(f.HealthCheckPath, app.HealthCheckPath)
	setStringIfNonEmpty(f.HealthCheckPort, app.HealthCheckPort)
	if app.LimitsMemorySwappiness != nil {
		*f.LimitsMemorySwappiness = types.Int64Value(*app.LimitsMemorySwappiness)
	}
	if app.LimitsCPUShares != nil {
		*f.LimitsCPUShares = types.Int64Value(*app.LimitsCPUShares)
	}
	if app.HealthCheckEnabled != nil {
		*f.HealthCheckEnabled = types.BoolValue(*app.HealthCheckEnabled)
	}
	if app.HealthCheckInterval != nil {
		*f.HealthCheckInterval = types.Int64Value(*app.HealthCheckInterval)
	}
	if app.HealthCheckTimeout != nil {
		*f.HealthCheckTimeout = types.Int64Value(*app.HealthCheckTimeout)
	}
	if app.HealthCheckRetries != nil {
		*f.HealthCheckRetries = types.Int64Value(*app.HealthCheckRetries)
	}
	if app.HealthCheckStartPeriod != nil {
		*f.HealthCheckStartPeriod = types.Int64Value(*app.HealthCheckStartPeriod)
	}
	if app.IsAutoDeployEnabled != nil {
		*f.IsAutoDeployEnabled = types.BoolValue(*app.IsAutoDeployEnabled)
	}
}

// buildUpdateInput constructs the shared UpdateApplicationInput from field pointers.
func buildUpdateInput(f commonAppFields) client.UpdateApplicationInput {
	strPtr := flex.StringValueOrNull
	int64Ptr := flex.Int64PtrFromFramework
	boolPtr := flex.BoolValueOrNull
	return client.UpdateApplicationInput{
		Name:               strPtr(*f.Name),
		Description:        strPtr(*f.Description),
		GitRepository:      strPtr(*f.GitRepository),
		GitBranch:          strPtr(*f.GitBranch),
		BuildPack:          strPtr(*f.BuildPack),
		PortsExposes:       strPtr(*f.PortsExposes),
		FQDN:               strPtr(*f.FQDN),
		DockerfileLocation: strPtr(*f.DockerfileLocation),
		InstallCommand:     strPtr(*f.InstallCommand),
		BuildCommand:       strPtr(*f.BuildCommand),
		StartCommand:       strPtr(*f.StartCommand),
		// Resource limits
		LimitsMemory:            strPtr(*f.LimitsMemory),
		LimitsMemorySwap:        strPtr(*f.LimitsMemorySwap),
		LimitsMemorySwappiness:  int64Ptr(*f.LimitsMemorySwappiness),
		LimitsMemoryReservation: strPtr(*f.LimitsMemoryReservation),
		LimitsCPUs:              strPtr(*f.LimitsCPUs),
		LimitsCPUSet:            strPtr(*f.LimitsCPUSet),
		LimitsCPUShares:         int64Ptr(*f.LimitsCPUShares),
		// Health checks
		HealthCheckEnabled:     boolPtr(*f.HealthCheckEnabled),
		HealthCheckPath:        strPtr(*f.HealthCheckPath),
		HealthCheckPort:        strPtr(*f.HealthCheckPort),
		HealthCheckInterval:    int64Ptr(*f.HealthCheckInterval),
		HealthCheckTimeout:     int64Ptr(*f.HealthCheckTimeout),
		HealthCheckRetries:     int64Ptr(*f.HealthCheckRetries),
		HealthCheckStartPeriod: int64Ptr(*f.HealthCheckStartPeriod),
		// Auto-deploy
		IsAutoDeployEnabled: boolPtr(*f.IsAutoDeployEnabled),
	}
}

// CommonAppAttrs returns the shared schema attributes for all application types.
func CommonAppAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the application.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"project_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"server_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"environment_name": schema.StringAttribute{
			MarkdownDescription: "The environment name for the application (defaults to `production`). Changing this forces a new resource.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("production"),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators:          []validator.String{validate.FQDN()},
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
			Computed:            true,
		},
		// Resource limits
		"limits_memory": schema.StringAttribute{
			MarkdownDescription: "Memory limit (e.g., `512m`, `2g`).",
			Optional:            true,
		},
		"limits_memory_swap": schema.StringAttribute{
			MarkdownDescription: "Memory swap limit (e.g., `1g`).",
			Optional:            true,
		},
		"limits_memory_swappiness": schema.Int64Attribute{
			MarkdownDescription: "Memory swappiness (0-100).",
			Optional:            true,
		},
		"limits_memory_reservation": schema.StringAttribute{
			MarkdownDescription: "Memory reservation (e.g., `256m`).",
			Optional:            true,
		},
		"limits_cpus": schema.StringAttribute{
			MarkdownDescription: "CPU limit (e.g., `0.5`, `2`).",
			Optional:            true,
		},
		"limits_cpuset": schema.StringAttribute{
			MarkdownDescription: "CPU set restriction (e.g., `0-3`, `0,2`).",
			Optional:            true,
		},
		"limits_cpu_shares": schema.Int64Attribute{
			MarkdownDescription: "CPU shares (relative weight).",
			Optional:            true,
		},
		// Health checks
		"health_check_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether health checks are enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"health_check_path": schema.StringAttribute{
			MarkdownDescription: "The URL path for health checks.",
			Optional:            true,
		},
		"health_check_port": schema.StringAttribute{
			MarkdownDescription: "The port for health checks.",
			Optional:            true,
		},
		"health_check_interval": schema.Int64Attribute{
			MarkdownDescription: "Health check interval in seconds.",
			Optional:            true,
		},
		"health_check_timeout": schema.Int64Attribute{
			MarkdownDescription: "Health check timeout in seconds.",
			Optional:            true,
		},
		"health_check_retries": schema.Int64Attribute{
			MarkdownDescription: "Number of health check retries.",
			Optional:            true,
		},
		"health_check_start_period": schema.Int64Attribute{
			MarkdownDescription: "Health check start period in seconds.",
			Optional:            true,
		},
		// Auto-deploy
		"is_auto_deploy_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto-deploy on push is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// updateAndReadBack performs the shared update-then-read pattern for all
// application resources.
func updateAndReadBack(
	ctx context.Context,
	c *client.Client,
	uuid string,
	input client.UpdateApplicationInput,
	resp *resource.UpdateResponse,
	flatten func(*client.Application),
) {
	if _, err := c.UpdateApplication(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after update", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	flatten(app)
}
