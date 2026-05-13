package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	// Extended health checks
	HealthCheckCommand      *types.String
	HealthCheckHost         *types.String
	HealthCheckMethod       *types.String
	HealthCheckResponseText *types.String
	HealthCheckReturnCode   *types.Int64
	HealthCheckScheme       *types.String
	HealthCheckType         *types.String
	// Auto-deploy
	IsAutoDeployEnabled *types.Bool
	// Extended build/deploy settings
	BaseDirectory                   *types.String
	Dockerfile                      *types.String
	DockerfileTargetBuild           *types.String
	DockerRegistryImageTag          *types.String
	DockerComposeLocation           *types.String
	DockerComposeCustomBuildCommand *types.String
	DockerComposeCustomStartCommand *types.String
	DockerComposeDomains            *types.String
	GitCommitSha                    *types.String
	PublishDirectory                *types.String
	WatchPaths                      *types.String
	PreviewURLTemplate              *types.String
	// Container/Network settings
	CustomDockerRunOptions   *types.String
	CustomLabels             *types.String
	CustomNetworkAliases     *types.String
	CustomNginxConfiguration *types.String
	PortsMappings            *types.String
	ConnectToDockerNetwork   *types.Bool
	// Redirect & static
	Redirect    *types.String
	StaticImage *types.String
	IsStatic    *types.Bool
	IsSPA       *types.Bool
	// Security & Auth
	IsForceHTTPSEnabled    *types.Bool
	IsHTTPBasicAuthEnabled *types.Bool
	HTTPBasicAuthUsername  *types.String
	HTTPBasicAuthPassword  *types.String
	// Deployment commands
	PreDeploymentCommand           *types.String
	PreDeploymentCommandContainer  *types.String
	PostDeploymentCommand          *types.String
	PostDeploymentCommandContainer *types.String
	// Webhook secrets
	ManualWebhookSecretBitbucket *types.String
	ManualWebhookSecretGitea     *types.String
	ManualWebhookSecretGitHub    *types.String
	ManualWebhookSecretGitLab    *types.String
	// Other settings
	ForceDomainOverride           *types.Bool
	IsContainerLabelEscapeEnabled *types.Bool
	IsPreserveRepositoryEnabled   *types.Bool
	UseBuildServer                *types.Bool
}

// flattenApplicationCommon maps shared API fields into any application model
// via field pointers. Nil pointers are skipped (allows partial models like
// compose or docker image to omit inapplicable fields).
func flattenApplicationCommon(app *client.Application, f commonAppFields) {
	*f.UUID = types.StringValue(app.UUID)
	*f.Name = types.StringValue(app.Name)
	*f.Description = flex.StringToFramework(app.Description)
	// Coolify normalizes GitHub URLs by stripping the "https://github.com/"
	// prefix (e.g. "https://github.com/org/repo" becomes "org/repo"). Preserve
	// the user's original input if the API value is a suffix of it.
	if f.GitRepository != nil {
		if !f.GitRepository.IsNull() && !f.GitRepository.IsUnknown() && strings.HasSuffix(f.GitRepository.ValueString(), app.GitRepository) {
			// keep user's original value
		} else {
			*f.GitRepository = types.StringValue(app.GitRepository)
		}
	}
	if f.GitBranch != nil {
		*f.GitBranch = types.StringValue(app.GitBranch)
	}
	if f.BuildPack != nil {
		*f.BuildPack = types.StringValue(app.BuildPack)
	}
	// Coolify may override ports_exposes (e.g. return 80 instead of 3000
	// for Dockerfile apps). Preserve the user's configured value.
	if f.PortsExposes != nil && app.PortsExposes != "" {
		if f.PortsExposes.IsNull() || f.PortsExposes.IsUnknown() {
			*f.PortsExposes = types.StringValue(app.PortsExposes)
		}
	}
	*f.FQDN = flex.StringToFramework(app.FQDN)
	// Coolify does not return dockerfile_location on GET for most app types.
	// Preserve the user's configured value to avoid "inconsistent result after apply".
	// The value IS sent on Create/Update, just not returned on read-back.
	if f.DockerfileLocation != nil && app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	if f.InstallCommand != nil {
		*f.InstallCommand = flex.StringToFramework(app.InstallCommand)
	}
	if f.BuildCommand != nil {
		*f.BuildCommand = flex.StringToFramework(app.BuildCommand)
	}
	if f.StartCommand != nil {
		*f.StartCommand = flex.StringToFramework(app.StartCommand)
	}
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
	flattenExtendedFields(app, f)
}

// flattenLimitsAndHealth sets resource limits, health checks, and auto-deploy
// fields from the API response. Extracted to keep flattenApplicationCommon
// under the gocognit complexity threshold.
func flattenLimitsAndHealth(app *client.Application, f commonAppFields) {
	// Only update optional fields when the user configured them (state is
	// not null/unknown). Coolify returns API defaults ("0", 30, etc.) for
	// unconfigured fields. Setting those would cause "Provider produced
	// inconsistent result after apply" because the plan has null but the
	// read-back would return a value.
	flex.SetStringIfConfigured(f.LimitsMemory, app.LimitsMemory)
	flex.SetStringIfConfigured(f.LimitsMemorySwap, app.LimitsMemorySwap)
	flex.SetStringIfConfigured(f.LimitsMemoryReservation, app.LimitsMemoryReservation)
	flex.SetStringIfConfigured(f.LimitsCPUs, app.LimitsCPUs)
	flex.SetStringIfConfigured(f.LimitsCPUSet, app.LimitsCPUSet)
	flex.SetStringIfConfigured(f.HealthCheckPath, app.HealthCheckPath)
	flex.SetStringIfConfigured(f.HealthCheckPort, app.HealthCheckPort)
	flex.SetInt64IfConfigured(f.LimitsMemorySwappiness, app.LimitsMemorySwappiness)
	flex.SetInt64IfConfigured(f.LimitsCPUShares, app.LimitsCPUShares)
	flex.SetInt64IfConfigured(f.HealthCheckInterval, app.HealthCheckInterval)
	flex.SetInt64IfConfigured(f.HealthCheckTimeout, app.HealthCheckTimeout)
	flex.SetInt64IfConfigured(f.HealthCheckRetries, app.HealthCheckRetries)
	flex.SetInt64IfConfigured(f.HealthCheckStartPeriod, app.HealthCheckStartPeriod)
	// Extended health check fields (optional, no defaults)
	flex.SetStringIfConfigured(f.HealthCheckCommand, app.HealthCheckCommand)
	flex.SetStringIfConfigured(f.HealthCheckResponseText, app.HealthCheckResponseText)
	// Extended health check fields with defaults (always set from API)
	*f.HealthCheckHost = flex.StringValueOrDefault(app.HealthCheckHost, "localhost")
	*f.HealthCheckMethod = flex.StringValueOrDefault(app.HealthCheckMethod, "GET")
	*f.HealthCheckScheme = flex.StringValueOrDefault(app.HealthCheckScheme, "http")
	*f.HealthCheckType = flex.StringValueOrDefault(app.HealthCheckType, "http")
	if app.HealthCheckReturnCode != nil {
		*f.HealthCheckReturnCode = types.Int64Value(*app.HealthCheckReturnCode)
	}
	// health_check_enabled and is_auto_deploy_enabled have schema defaults
	// (Computed+Default), so they are never null in plan. Safe to always set.
	if app.HealthCheckEnabled != nil {
		*f.HealthCheckEnabled = types.BoolValue(*app.HealthCheckEnabled)
	}
	if app.IsAutoDeployEnabled != nil {
		*f.IsAutoDeployEnabled = types.BoolValue(*app.IsAutoDeployEnabled)
	}
}

// flattenExtendedFields sets extended application fields from the API response.
// Extracted to keep flattenApplicationCommon under the gocognit complexity threshold.
func flattenExtendedFields(app *client.Application, f commonAppFields) {
	// Optional string fields (set only if user configured them)
	flex.SetStringIfConfigured(f.BaseDirectory, app.BaseDirectory)
	flex.SetStringIfConfigured(f.PublishDirectory, app.PublishDirectory)
	flex.SetStringIfConfigured(f.Dockerfile, app.Dockerfile)
	flex.SetStringIfConfigured(f.DockerRegistryImageTag, app.DockerRegistryImageTag)
	flex.SetStringIfConfigured(f.DockerComposeDomains, app.DockerComposeDomains)
	flex.SetStringIfConfigured(f.GitCommitSha, app.GitCommitSha)
	flex.SetStringIfConfigured(f.WatchPaths, app.WatchPaths)
	flex.SetStringIfConfigured(f.CustomDockerRunOptions, app.CustomDockerRunOptions)
	flex.SetStringIfConfigured(f.CustomLabels, app.CustomLabels)
	flex.SetStringIfConfigured(f.CustomNetworkAliases, app.CustomNetworkAliases)
	flex.SetStringIfConfigured(f.CustomNginxConfiguration, app.CustomNginxConfiguration)
	flex.SetStringIfConfigured(f.PortsMappings, app.PortsMappings)
	flex.SetStringIfConfigured(f.HTTPBasicAuthUsername, app.HTTPBasicAuthUsername)
	flex.SetStringIfConfigured(f.HTTPBasicAuthPassword, app.HTTPBasicAuthPassword)
	flex.SetStringIfConfigured(f.PreDeploymentCommand, app.PreDeploymentCommand)
	flex.SetStringIfConfigured(f.PreDeploymentCommandContainer, app.PreDeploymentCommandContainer)
	flex.SetStringIfConfigured(f.PostDeploymentCommand, app.PostDeploymentCommand)
	flex.SetStringIfConfigured(f.PostDeploymentCommandContainer, app.PostDeploymentCommandContainer)
	// Nil-safe optional string fields (resource-specific extras)
	if f.DockerfileTargetBuild != nil {
		flex.SetStringIfConfigured(f.DockerfileTargetBuild, app.DockerfileTargetBuild)
	}
	if f.DockerComposeLocation != nil {
		flex.SetStringIfConfigured(f.DockerComposeLocation, app.DockerComposeLocation)
	}
	if f.DockerComposeCustomBuildCommand != nil {
		flex.SetStringIfConfigured(f.DockerComposeCustomBuildCommand, app.DockerComposeCustomBuildCommand)
	}
	if f.DockerComposeCustomStartCommand != nil {
		flex.SetStringIfConfigured(f.DockerComposeCustomStartCommand, app.DockerComposeCustomStartCommand)
	}
	flattenExtendedDefaults(app, f)
}

// flattenExtendedDefaults sets fields with Computed+Default and sensitive fields.
func flattenExtendedDefaults(app *client.Application, f commonAppFields) {
	// Computed+Default string fields (always set from API)
	setStr := func(dst *types.String, v types.String) {
		if dst != nil {
			*dst = v
		}
	}
	setStr(f.Redirect, flex.StringValueOrDefault(app.Redirect, "both"))
	setStr(f.StaticImage, flex.StringValueOrDefault(app.StaticImage, "nginx:alpine"))
	// Computed+Default+Sensitive fields (server-generated, always set)
	setStr(f.PreviewURLTemplate, flex.StringToFramework(app.PreviewURLTemplate))
	setStr(f.ManualWebhookSecretBitbucket, flex.StringToFramework(app.ManualWebhookSecretBitbucket))
	setStr(f.ManualWebhookSecretGitea, flex.StringToFramework(app.ManualWebhookSecretGitea))
	setStr(f.ManualWebhookSecretGitHub, flex.StringToFramework(app.ManualWebhookSecretGitHub))
	setStr(f.ManualWebhookSecretGitLab, flex.StringToFramework(app.ManualWebhookSecretGitLab))
	// Computed+Default bool fields (always set from API)
	setBoolDefault := func(dst *types.Bool, v *bool, def bool) {
		if dst == nil {
			return
		}
		if v != nil {
			*dst = types.BoolValue(*v)
		} else {
			*dst = types.BoolValue(def)
		}
	}
	setBoolDefault(f.ConnectToDockerNetwork, app.ConnectToDockerNetwork, false)
	setBoolDefault(f.IsHTTPBasicAuthEnabled, app.IsHTTPBasicAuthEnabled, false)
	setBoolDefault(f.IsStatic, app.IsStatic, false)
	setBoolDefault(f.IsSPA, app.IsSPA, false)
	setBoolDefault(f.IsForceHTTPSEnabled, app.IsForceHTTPSEnabled, true)
	setBoolDefault(f.IsContainerLabelEscapeEnabled, app.IsContainerLabelEscapeEnabled, true)
	setBoolDefault(f.IsPreserveRepositoryEnabled, app.IsPreserveRepositoryEnabled, false)
	setBoolDefault(f.UseBuildServer, app.UseBuildServer, false)
	// Optional bool fields (no default)
	if f.ForceDomainOverride != nil && app.ForceDomainOverride != nil {
		if !f.ForceDomainOverride.IsNull() && !f.ForceDomainOverride.IsUnknown() {
			*f.ForceDomainOverride = types.BoolValue(*app.ForceDomainOverride)
		}
	}
}

// buildUpdateInput constructs the shared UpdateApplicationInput from field pointers.
func buildUpdateInput(f commonAppFields) client.UpdateApplicationInput {
	input := buildCoreUpdateFields(f)
	addExtendedUpdateFields(f, &input)
	return input
}

// buildCoreUpdateFields populates the core UpdateApplicationInput fields.
func buildCoreUpdateFields(f commonAppFields) client.UpdateApplicationInput {
	strPtr := flex.StringValueOrNull
	int64Ptr := flex.Int64PtrFromFramework
	boolPtr := flex.BoolValueOrNull
	input := client.UpdateApplicationInput{
		Name:        strPtr(*f.Name),
		Description: strPtr(*f.Description),
		FQDN:        strPtr(*f.FQDN),
		// Resource limits
		LimitsMemory:            strPtr(*f.LimitsMemory),
		LimitsMemorySwap:        strPtr(*f.LimitsMemorySwap),
		LimitsMemorySwappiness:  int64Ptr(*f.LimitsMemorySwappiness),
		LimitsMemoryReservation: strPtr(*f.LimitsMemoryReservation),
		LimitsCPUs:              strPtr(*f.LimitsCPUs),
		LimitsCPUSet:            strPtr(*f.LimitsCPUSet),
		LimitsCPUShares:         int64Ptr(*f.LimitsCPUShares),
		// Health checks
		HealthCheckEnabled:      boolPtr(*f.HealthCheckEnabled),
		HealthCheckPath:         strPtr(*f.HealthCheckPath),
		HealthCheckPort:         strPtr(*f.HealthCheckPort),
		HealthCheckInterval:     int64Ptr(*f.HealthCheckInterval),
		HealthCheckTimeout:      int64Ptr(*f.HealthCheckTimeout),
		HealthCheckRetries:      int64Ptr(*f.HealthCheckRetries),
		HealthCheckStartPeriod:  int64Ptr(*f.HealthCheckStartPeriod),
		HealthCheckCommand:      strPtr(*f.HealthCheckCommand),
		HealthCheckHost:         strPtr(*f.HealthCheckHost),
		HealthCheckMethod:       strPtr(*f.HealthCheckMethod),
		HealthCheckResponseText: strPtr(*f.HealthCheckResponseText),
		HealthCheckReturnCode:   int64Ptr(*f.HealthCheckReturnCode),
		HealthCheckScheme:       strPtr(*f.HealthCheckScheme),
		HealthCheckType:         strPtr(*f.HealthCheckType),
		// Auto-deploy
		IsAutoDeployEnabled: boolPtr(*f.IsAutoDeployEnabled),
	}
	// Nil-safe fields (not present in all resource models)
	if f.GitRepository != nil {
		input.GitRepository = strPtr(*f.GitRepository)
	}
	if f.GitBranch != nil {
		input.GitBranch = strPtr(*f.GitBranch)
	}
	if f.BuildPack != nil {
		input.BuildPack = strPtr(*f.BuildPack)
	}
	if f.PortsExposes != nil {
		input.PortsExposes = strPtr(*f.PortsExposes)
	}
	if f.InstallCommand != nil {
		input.InstallCommand = strPtr(*f.InstallCommand)
	}
	if f.BuildCommand != nil {
		input.BuildCommand = strPtr(*f.BuildCommand)
	}
	if f.StartCommand != nil {
		input.StartCommand = strPtr(*f.StartCommand)
	}
	if f.DockerfileLocation != nil {
		input.DockerfileLocation = strPtr(*f.DockerfileLocation)
	}
	return input
}

// addExtendedUpdateFields adds extended fields to an UpdateApplicationInput.
func addExtendedUpdateFields(f commonAppFields, input *client.UpdateApplicationInput) {
	strPtr := flex.StringValueOrNull
	boolPtr := flex.BoolValueOrNull
	// Build/deploy
	input.BaseDirectory = strPtr(*f.BaseDirectory)
	input.PublishDirectory = strPtr(*f.PublishDirectory)
	input.DockerRegistryImageTag = strPtr(*f.DockerRegistryImageTag)
	input.DockerComposeDomains = strPtr(*f.DockerComposeDomains)
	input.GitCommitSha = strPtr(*f.GitCommitSha)
	input.WatchPaths = strPtr(*f.WatchPaths)
	// preview_url_template is not in Coolify v4's update $allowedFields.
	// Container/Network
	input.CustomDockerRunOptions = strPtr(*f.CustomDockerRunOptions)
	input.CustomLabels = strPtr(*f.CustomLabels)
	input.CustomNetworkAliases = strPtr(*f.CustomNetworkAliases)
	input.CustomNginxConfiguration = strPtr(*f.CustomNginxConfiguration)
	input.PortsMappings = strPtr(*f.PortsMappings)
	// Redirect & static
	input.Redirect = strPtr(*f.Redirect)
	input.StaticImage = strPtr(*f.StaticImage)
	input.IsStatic = boolPtr(*f.IsStatic)
	input.IsSPA = boolPtr(*f.IsSPA)
	// Security & Auth
	input.IsForceHTTPSEnabled = boolPtr(*f.IsForceHTTPSEnabled)
	input.IsHTTPBasicAuthEnabled = boolPtr(*f.IsHTTPBasicAuthEnabled)
	input.HTTPBasicAuthUsername = strPtr(*f.HTTPBasicAuthUsername)
	input.HTTPBasicAuthPassword = strPtr(*f.HTTPBasicAuthPassword)
	// Deployment commands
	input.PreDeploymentCommand = strPtr(*f.PreDeploymentCommand)
	input.PreDeploymentCommandContainer = strPtr(*f.PreDeploymentCommandContainer)
	input.PostDeploymentCommand = strPtr(*f.PostDeploymentCommand)
	input.PostDeploymentCommandContainer = strPtr(*f.PostDeploymentCommandContainer)
	// Webhook secrets
	input.ManualWebhookSecretBitbucket = strPtr(*f.ManualWebhookSecretBitbucket)
	input.ManualWebhookSecretGitea = strPtr(*f.ManualWebhookSecretGitea)
	input.ManualWebhookSecretGitHub = strPtr(*f.ManualWebhookSecretGitHub)
	input.ManualWebhookSecretGitLab = strPtr(*f.ManualWebhookSecretGitLab)
	// Other settings
	input.ConnectToDockerNetwork = boolPtr(*f.ConnectToDockerNetwork)
	input.IsContainerLabelEscapeEnabled = boolPtr(*f.IsContainerLabelEscapeEnabled)
	input.IsPreserveRepositoryEnabled = boolPtr(*f.IsPreserveRepositoryEnabled)
	input.UseBuildServer = boolPtr(*f.UseBuildServer)
	// Nil-safe resource-specific fields
	if f.ForceDomainOverride != nil {
		input.ForceDomainOverride = boolPtr(*f.ForceDomainOverride)
	}
	if f.DockerfileTargetBuild != nil {
		input.DockerfileTargetBuild = strPtr(*f.DockerfileTargetBuild)
	}
	if f.DockerComposeLocation != nil {
		input.DockerComposeLocation = strPtr(*f.DockerComposeLocation)
	}
	if f.DockerComposeCustomBuildCommand != nil {
		input.DockerComposeCustomBuildCommand = strPtr(*f.DockerComposeCustomBuildCommand)
	}
	if f.DockerComposeCustomStartCommand != nil {
		input.DockerComposeCustomStartCommand = strPtr(*f.DockerComposeCustomStartCommand)
	}
}

// CommonAppAttrs returns the shared schema attributes for all application types.
func CommonAppAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := coreAppAttrs(ctx)
	mergeAttrs(attrs, extendedBuildDeployAttrs())
	mergeAttrs(attrs, extendedHealthCheckAttrs())
	mergeAttrs(attrs, securityNetworkAttrs())
	mergeAttrs(attrs, extra)
	return attrs
}

func mergeAttrs(dst, src map[string]schema.Attribute) {
	for k, v := range src {
		dst[k] = v
	}
}

// coreAppAttrs returns the core schema attributes (identity, status, limits,
// existing health checks, auto-deploy).
func coreAppAttrs(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
}

// extendedBuildDeployAttrs returns schema attributes for build, deploy, and static settings.
func extendedBuildDeployAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base_directory": schema.StringAttribute{
			MarkdownDescription: "The base directory for the application source code.",
			Optional:            true,
		},
		"publish_directory": schema.StringAttribute{
			MarkdownDescription: "The directory to publish for static sites.",
			Optional:            true,
		},
		"dockerfile": schema.StringAttribute{
			MarkdownDescription: "Inline Dockerfile content (base64 encoded).",
			Optional:            true,
		},
		"docker_registry_image_tag": schema.StringAttribute{
			MarkdownDescription: "The Docker registry image tag.",
			Optional:            true,
		},
		"docker_compose_domains": schema.StringAttribute{
			MarkdownDescription: "Domain mappings for Docker Compose services.",
			Optional:            true,
		},
		"git_commit_sha": schema.StringAttribute{
			MarkdownDescription: "The specific Git commit SHA to deploy.",
			Optional:            true,
		},
		"watch_paths": schema.StringAttribute{
			MarkdownDescription: "Paths to watch for changes (triggers auto-deploy).",
			Optional:            true,
		},
		"redirect": schema.StringAttribute{
			MarkdownDescription: "Domain redirect mode. Valid values: `www`, `non-www`, `both`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("both"),
			Validators:          []validator.String{stringvalidator.OneOf("www", "non-www", "both")},
		},
		"static_image": schema.StringAttribute{
			MarkdownDescription: "The Docker image to use for serving static sites.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("nginx:alpine"),
		},
		"is_static": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a static site.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_spa": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a single-page application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_preserve_repository_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to preserve the full Git repository (instead of shallow clone).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"use_build_server": schema.BoolAttribute{
			MarkdownDescription: "Whether to use a build server for building the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"preview_url_template": schema.StringAttribute{
			MarkdownDescription: "The URL template for preview deployments.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"pre_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run before deployment.",
			Optional:            true,
		},
		"pre_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the pre-deployment command in.",
			Optional:            true,
		},
		"post_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run after deployment.",
			Optional:            true,
		},
		"post_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the post-deployment command in.",
			Optional:            true,
		},
	}
}

// extendedHealthCheckAttrs returns schema attributes for extended health check settings.
func extendedHealthCheckAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"health_check_command": schema.StringAttribute{
			MarkdownDescription: "Custom health check command (used when type is `cmd`).",
			Optional:            true,
		},
		"health_check_host": schema.StringAttribute{
			MarkdownDescription: "The host for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("localhost"),
		},
		"health_check_method": schema.StringAttribute{
			MarkdownDescription: "The HTTP method for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("GET"),
			Validators:          []validator.String{stringvalidator.OneOf("GET", "HEAD", "POST", "OPTIONS")},
		},
		"health_check_response_text": schema.StringAttribute{
			MarkdownDescription: "Expected response text for health check validation.",
			Optional:            true,
		},
		"health_check_return_code": schema.Int64Attribute{
			MarkdownDescription: "Expected HTTP return code for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(200),
		},
		"health_check_scheme": schema.StringAttribute{
			MarkdownDescription: "The URL scheme for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("http"),
			Validators:          []validator.String{stringvalidator.OneOf("http", "https")},
		},
		"health_check_type": schema.StringAttribute{
			MarkdownDescription: "The type of health check. Valid values: `http`, `cmd`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("http"),
			Validators:          []validator.String{stringvalidator.OneOf("http", "cmd")},
		},
	}
}

// securityNetworkAttrs returns schema attributes for network, security, auth, and webhook settings.
func securityNetworkAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"connect_to_docker_network": schema.BoolAttribute{
			MarkdownDescription: "Whether to connect the application to the Docker network.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"custom_docker_run_options": schema.StringAttribute{
			MarkdownDescription: "Custom Docker run options passed to the container.",
			Optional:            true,
		},
		"custom_labels": schema.StringAttribute{
			MarkdownDescription: "Custom Docker labels for the container, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"custom_network_aliases": schema.StringAttribute{
			MarkdownDescription: "Custom network aliases for the container.",
			Optional:            true,
		},
		"custom_nginx_configuration": schema.StringAttribute{
			MarkdownDescription: "Custom Nginx configuration for the application, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"ports_mappings": schema.StringAttribute{
			MarkdownDescription: "Port mappings in `host:container` format, comma-separated (e.g. `8080:80` or `8080:80,8443:443`).",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(regexp.MustCompile(`^\d+:\d+(,\d+:\d+)*$`), "must be comma-separated host:container port pairs (e.g. \"8080:80\" or \"8080:80,8443:443\")"),
			},
		},
		"force_domain_override": schema.BoolAttribute{
			MarkdownDescription: "Whether to force domain override.",
			Optional:            true,
		},
		"is_container_label_escape_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether container label escaping is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_force_https_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to force HTTPS for the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_http_basic_auth_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether HTTP Basic Authentication is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"http_basic_auth_username": schema.StringAttribute{
			MarkdownDescription: "Username for HTTP Basic Authentication.",
			Optional:            true,
		},
		"http_basic_auth_password": schema.StringAttribute{
			MarkdownDescription: "Password for HTTP Basic Authentication.",
			Optional:            true,
			Sensitive:           true,
		},
		"manual_webhook_secret_bitbucket": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Bitbucket.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitea": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Gitea.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_github": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitHub.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitlab": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitLab.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
}

// setImportDefaults sets the default values for Computed+Default attributes
// during import. These must be set explicitly because Terraform does not apply
// schema defaults during import; the Read method relies on these initial values
// to avoid null-vs-default conflicts.
func setImportDefaults(ctx context.Context, resp *resource.ImportStateResponse) {
	set := func(attr string, v interface{}) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), v)...)
	}
	set("health_check_enabled", false)
	set("is_auto_deploy_enabled", true)
	set("redirect", "both")
	set("health_check_type", "http")
	set("health_check_method", "GET")
	set("health_check_scheme", "http")
	set("health_check_return_code", int64(200))
	set("health_check_host", "localhost")
	set("static_image", "nginx:alpine")
	set("connect_to_docker_network", false)
	set("is_http_basic_auth_enabled", false)
	set("is_static", false)
	set("is_spa", false)
	set("is_force_https_enabled", true)
	set("is_container_label_escape_enabled", true)
	set("is_preserve_repository_enabled", false)
	set("use_build_server", false)
}

// readBackAfterCreate reads the application after creation and handles the
// 404-on-readback case (server not SSH-reachable). Returns nil if an error
// was added to diagnostics.
func readBackAfterCreate(ctx context.Context, c *client.Client, uuid string, resp *resource.CreateResponse) *client.Application {
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			resp.Diagnostics.AddError(
				"Application created but not persisted",
				fmt.Sprintf("Coolify returned UUID %s but the application was not found on read-back. "+
					"This usually means the target server is not reachable via SSH. "+
					"Verify the server is online and SSH-accessible before retrying.", uuid),
			)
			return nil
		}
		resp.Diagnostics.AddError("Error reading application after creation", fmt.Sprintf("application %s: %s", uuid, err))
		return nil
	}
	return app
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
