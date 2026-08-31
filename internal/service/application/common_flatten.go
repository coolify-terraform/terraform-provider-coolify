package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// flattenApplicationCommon maps shared API fields into any application model
// via field pointers. Nil pointers are skipped (allows partial models like
// compose or docker image to omit inapplicable fields).
func flattenApplicationCommon(app *client.Application, f commonAppFields) {
	*f.UUID = types.StringValue(app.UUID)
	*f.Name = types.StringValue(app.Name)
	*f.Description = flex.StringToFramework(app.Description)
	if f.GitRepository != nil {
		*f.GitRepository = resolveGitRepository(*f.GitRepository, app.GitRepository)
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
	// Empty FQDN is a real state (internal apps / cleared domains), not "unset".
	// Prefer "" over null so clear and autogenerate_domain=false round-trip cleanly.
	if app.Domains == "" {
		*f.Domains = types.StringValue("")
	} else {
		*f.Domains = types.StringValue(app.Domains)
	}
	// Coolify does not return dockerfile_location on GET for most app types.
	// Preserve the user's configured value to avoid "inconsistent result after apply".
	// The value IS sent on Create/Update, just not returned on read-back.
	if f.DockerfileLocation != nil && app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	flex.SetStringSeedOrClear(f.InstallCommand, app.InstallCommand)
	// Seed null state from API so import populates build/start commands (#577).
	flex.SetStringSeedOrClear(f.BuildCommand, app.BuildCommand)
	flex.SetStringSeedOrClear(f.StartCommand, app.StartCommand)
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
	// redeploy_on_update is a Terraform-only flag not returned by the API.
	// Preserve the existing state value; default to false on import.
	if f.RedeployOnUpdate != nil {
		if f.RedeployOnUpdate.IsNull() || f.RedeployOnUpdate.IsUnknown() {
			*f.RedeployOnUpdate = types.BoolValue(false)
		}
	}
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
	flex.SetStringOrClear(f.LimitsCPUSet, app.LimitsCPUSet)
	flex.SetStringIfConfigured(f.HealthCheckPath, app.HealthCheckPath)
	flex.SetStringOrClear(f.HealthCheckPort, app.HealthCheckPort)
	flex.SetInt64IfConfigured(f.LimitsMemorySwappiness, app.LimitsMemorySwappiness)
	flex.SetInt64IfConfigured(f.LimitsCPUShares, app.LimitsCPUShares)
	flex.SetInt64IfConfigured(f.HealthCheckInterval, app.HealthCheckInterval)
	flex.SetInt64IfConfigured(f.HealthCheckTimeout, app.HealthCheckTimeout)
	flex.SetInt64IfConfigured(f.HealthCheckRetries, app.HealthCheckRetries)
	flex.SetInt64IfConfigured(f.HealthCheckStartPeriod, app.HealthCheckStartPeriod)
	// Extended health check fields (optional, no defaults)
	flex.SetStringOrClear(f.HealthCheckCommand, app.HealthCheckCommand)
	flex.SetStringOrClear(f.HealthCheckResponseText, app.HealthCheckResponseText)
	// Extended health check fields with defaults (always set from API)
	*f.HealthCheckHost = flex.StringValueOrDefault(app.HealthCheckHost, defaultHealthCheckHost)
	*f.HealthCheckMethod = flex.StringValueOrDefault(app.HealthCheckMethod, defaultHealthCheckMeth)
	*f.HealthCheckScheme = flex.StringValueOrDefault(app.HealthCheckScheme, defaultHealthCheckSchm)
	*f.HealthCheckType = flex.StringValueOrDefault(app.HealthCheckType, defaultHealthCheckType)
	if app.HealthCheckReturnCode != nil {
		*f.HealthCheckReturnCode = types.Int64Value(*app.HealthCheckReturnCode)
	}
	// health_check_enabled and is_auto_deploy_enabled are Optional+Computed
	// without Default. Always set them to resolve unknown values after Create.
	// When API returns nil, use the Coolify DB default.
	if app.HealthCheckEnabled != nil {
		*f.HealthCheckEnabled = types.BoolValue(*app.HealthCheckEnabled)
	} else {
		*f.HealthCheckEnabled = types.BoolValue(false)
	}
	if app.IsAutoDeployEnabled != nil {
		*f.IsAutoDeployEnabled = types.BoolValue(*app.IsAutoDeployEnabled)
	}
}

// flattenExtendedFields sets extended application fields from the API response.
// Extracted to keep flattenApplicationCommon under the gocognit complexity threshold.
func flattenExtendedFields(app *client.Application, f commonAppFields) {
	// base_directory defaults to "/" in Coolify. Seed non-default values into
	// null state (import) but do not force "/" onto omitted create plans (#577).
	flex.SetStringSeedIfConfigured(f.BaseDirectory, app.BaseDirectory, "/")
	flex.SetStringIfConfigured(f.GitCommitSha, app.GitCommitSha)
	// custom_labels and custom_nginx_configuration: API requires base64 input,
	// stores base64, and returns base64 on GET (with read:sensitive). Users write
	// raw content; provider auto-encodes via EnsureBase64. ResolveBase64Field
	// preserves the user's raw value when it matches the API's base64, avoiding
	// perpetual diffs. Also handles pre-encoded input for backward compatibility.
	if f.CustomLabels != nil {
		*f.CustomLabels = flex.ResolveBase64Field(*f.CustomLabels, app.CustomLabels)
	}
	if f.CustomNginxConfiguration != nil {
		*f.CustomNginxConfiguration = flex.ResolveBase64Field(*f.CustomNginxConfiguration, app.CustomNginxConfiguration)
	}
	// Nullable fields — seed null state from API (import) and clear when the
	// API returns empty for configured values (UI drift).
	flex.SetStringSeedOrClear(f.PublishDirectory, app.PublishDirectory)
	flex.SetStringIfConfigured(f.Dockerfile, app.Dockerfile)
	flex.SetStringOrClear(f.DockerRegistryImageTag, app.DockerRegistryImageTag)
	// Coolify stores/returns an object map; Terraform config uses the write
	// array shape. Normalize on read so plan stays empty (#652).
	resolveDockerComposeDomains(f.DockerComposeDomains, app.DockerComposeDomains.String())
	flex.SetStringSeedOrClear(f.WatchPaths, app.WatchPaths)
	flex.SetStringOrClear(f.CustomDockerRunOptions, app.CustomDockerRunOptions)
	flex.SetStringOrClear(f.CustomNetworkAliases, app.CustomNetworkAliases)
	flex.SetStringOrClear(f.PortsMappings, app.PortsMappings)
	flex.SetStringIfConfigured(f.HTTPBasicAuthUsername, app.HTTPBasicAuthUsername)
	flex.SetStringIfConfigured(f.HTTPBasicAuthPassword, app.HTTPBasicAuthPassword)
	flex.SetStringOrClear(f.PreDeploymentCommand, app.PreDeploymentCommand)
	flex.SetStringOrClear(f.PreDeploymentCommandContainer, app.PreDeploymentCommandContainer)
	flex.SetStringOrClear(f.PostDeploymentCommand, app.PostDeploymentCommand)
	flex.SetStringOrClear(f.PostDeploymentCommandContainer, app.PostDeploymentCommandContainer)
	// Nil-safe optional string fields (resource-specific extras, all nullable)
	if f.DockerfileTargetBuild != nil {
		flex.SetStringOrClear(f.DockerfileTargetBuild, app.DockerfileTargetBuild)
	}
	if f.DockerComposeLocation != nil {
		flex.SetStringIfConfigured(f.DockerComposeLocation, app.DockerComposeLocation)
	}
	if f.DockerComposeCustomBuildCommand != nil {
		flex.SetStringOrClear(f.DockerComposeCustomBuildCommand, app.DockerComposeCustomBuildCommand)
	}
	if f.DockerComposeCustomStartCommand != nil {
		flex.SetStringOrClear(f.DockerComposeCustomStartCommand, app.DockerComposeCustomStartCommand)
	}
	flattenExtendedDefaults(app, f)
}

// flattenExtendedDefaults sets fields with Computed+Default and sensitive fields.
func flattenExtendedDefaults(app *client.Application, f commonAppFields) {
	// Computed+Default string fields (always set from API)
	setString := func(dst *types.String, v types.String) {
		if dst == nil {
			return
		}
		*dst = v
	}
	setString(f.Redirect, flex.StringValueOrDefault(app.Redirect, defaultRedirect))
	setString(f.StaticImage, flex.StringValueOrDefault(app.StaticImage, defaultStaticImage))
	// Computed+Default+Sensitive fields (server-generated, always set)
	setString(f.PreviewURLTemplate, flex.StringToFramework(app.PreviewURLTemplate))
	// Webhook secrets are encrypted and hidden without root/read:sensitive.
	// Preserve planned/state values when GET returns empty (#575).
	flex.SetStringPreserveEmpty(f.ManualWebhookSecretBitbucket, app.ManualWebhookSecretBitbucket)
	flex.SetStringPreserveEmpty(f.ManualWebhookSecretGitea, app.ManualWebhookSecretGitea)
	flex.SetStringPreserveEmpty(f.ManualWebhookSecretGitHub, app.ManualWebhookSecretGitHub)
	flex.SetStringPreserveEmpty(f.ManualWebhookSecretGitLab, app.ManualWebhookSecretGitLab)
	// Computed+Default bool fields (always set from API)
	setBoolDefault(f.ConnectToDockerNetwork, app.ConnectToDockerNetwork, false)
	setBoolDefault(f.IsHTTPBasicAuthEnabled, app.IsHTTPBasicAuthEnabled, false)
	setBoolDefault(f.IsStatic, app.IsStatic, false)
	setBoolDefault(f.IsSPA, app.IsSPA, false)
	setBoolDefault(f.IsForceHTTPSEnabled, app.IsForceHTTPSEnabled, true)
	setBoolDefault(f.IsContainerLabelEscapeEnabled, app.IsContainerLabelEscapeEnabled, true)
	setBoolDefault(f.IsPreserveRepositoryEnabled, app.IsPreserveRepositoryEnabled, false)
	setBoolDefault(f.UseBuildServer, app.UseBuildServer, false)
	setBoolDefault(f.IsPreviewDeploymentsEnabled, app.IsPreviewDeploymentsEnabled, false)
	setBoolDefault(f.UseBuildSecrets, app.UseBuildSecrets, false)
	if f.StopGracePeriod != nil {
		if app.StopGracePeriod != nil {
			*f.StopGracePeriod = types.Int64Value(*app.StopGracePeriod)
		} else if f.StopGracePeriod.IsNull() || f.StopGracePeriod.IsUnknown() {
			// Preserve null when API omits; UseStateForUnknown covers plan.
			*f.StopGracePeriod = types.Int64Null()
		}
	}
	flattenApplicationSettingFields(app, f)
	flattenRestartLimitFields(app, f)
	// instant_deploy is create-only and never returned by the API.
	// Preserve state value when set; default to false otherwise (import).
	if f.InstantDeploy != nil && (f.InstantDeploy.IsNull() || f.InstantDeploy.IsUnknown()) {
		*f.InstantDeploy = types.BoolValue(false)
	}
	// Create-only: API never returns autogenerate_domain; preserve plan/state, default true on import.
	if f.AutogenerateDomain != nil && (f.AutogenerateDomain.IsNull() || f.AutogenerateDomain.IsUnknown()) {
		*f.AutogenerateDomain = types.BoolValue(true)
	}
	// Optional bool fields (no default)
	if f.ForceDomainOverride != nil && app.ForceDomainOverride != nil {
		if !f.ForceDomainOverride.IsNull() && !f.ForceDomainOverride.IsUnknown() {
			*f.ForceDomainOverride = types.BoolValue(*app.ForceDomainOverride)
		}
	}
}

// buildUpdateInput constructs the shared UpdateApplicationInput from field pointers,
// only including fields that differ between plan and state.
func buildUpdateInput(plan, state commonAppFields) client.UpdateApplicationInput {
	input := buildCoreUpdateFields(plan, state)
	addExtendedUpdateFields(plan, state, &input)
	return input
}

// buildCoreUpdateFields populates the core UpdateApplicationInput fields,
// only including fields that differ between plan and state.
func buildCoreUpdateFields(plan, state commonAppFields) client.UpdateApplicationInput {
	strDiff := flex.StringIfChanged
	int64Diff := flex.Int64IfChanged
	boolDiff := flex.BoolIfChanged
	input := client.UpdateApplicationInput{
		Name:        strDiff(*plan.Name, *state.Name),
		Description: strDiff(*plan.Description, *state.Description),
		Domains:     strDiff(*plan.Domains, *state.Domains),
		// Resource limits
		LimitsMemory:            strDiff(*plan.LimitsMemory, *state.LimitsMemory),
		LimitsMemorySwap:        strDiff(*plan.LimitsMemorySwap, *state.LimitsMemorySwap),
		LimitsMemorySwappiness:  int64Diff(*plan.LimitsMemorySwappiness, *state.LimitsMemorySwappiness),
		LimitsMemoryReservation: strDiff(*plan.LimitsMemoryReservation, *state.LimitsMemoryReservation),
		LimitsCPUs:              strDiff(*plan.LimitsCPUs, *state.LimitsCPUs),
		LimitsCPUSet:            strDiff(*plan.LimitsCPUSet, *state.LimitsCPUSet),
		LimitsCPUShares:         int64Diff(*plan.LimitsCPUShares, *state.LimitsCPUShares),
		// Health checks
		HealthCheckEnabled:      boolDiff(*plan.HealthCheckEnabled, *state.HealthCheckEnabled),
		HealthCheckPath:         strDiff(*plan.HealthCheckPath, *state.HealthCheckPath),
		HealthCheckPort:         strDiff(*plan.HealthCheckPort, *state.HealthCheckPort),
		HealthCheckInterval:     int64Diff(*plan.HealthCheckInterval, *state.HealthCheckInterval),
		HealthCheckTimeout:      int64Diff(*plan.HealthCheckTimeout, *state.HealthCheckTimeout),
		HealthCheckRetries:      int64Diff(*plan.HealthCheckRetries, *state.HealthCheckRetries),
		HealthCheckStartPeriod:  int64Diff(*plan.HealthCheckStartPeriod, *state.HealthCheckStartPeriod),
		HealthCheckCommand:      strDiff(*plan.HealthCheckCommand, *state.HealthCheckCommand),
		HealthCheckHost:         strDiff(*plan.HealthCheckHost, *state.HealthCheckHost),
		HealthCheckMethod:       strDiff(*plan.HealthCheckMethod, *state.HealthCheckMethod),
		HealthCheckResponseText: strDiff(*plan.HealthCheckResponseText, *state.HealthCheckResponseText),
		HealthCheckReturnCode:   int64Diff(*plan.HealthCheckReturnCode, *state.HealthCheckReturnCode),
		HealthCheckScheme:       strDiff(*plan.HealthCheckScheme, *state.HealthCheckScheme),
		HealthCheckType:         strDiff(*plan.HealthCheckType, *state.HealthCheckType),
		// Auto-deploy
		IsAutoDeployEnabled: boolDiff(*plan.IsAutoDeployEnabled, *state.IsAutoDeployEnabled),
	}
	// Nil-safe fields (not present in all resource models)
	if plan.GitRepository != nil && state.GitRepository != nil {
		input.GitRepository = strDiff(*plan.GitRepository, *state.GitRepository)
	}
	if plan.GitBranch != nil && state.GitBranch != nil {
		input.GitBranch = strDiff(*plan.GitBranch, *state.GitBranch)
	}
	if plan.BuildPack != nil && state.BuildPack != nil {
		input.BuildPack = strDiff(*plan.BuildPack, *state.BuildPack)
	}
	if plan.PortsExposes != nil && state.PortsExposes != nil {
		input.PortsExposes = strDiff(*plan.PortsExposes, *state.PortsExposes)
	}
	if plan.InstallCommand != nil && state.InstallCommand != nil {
		input.InstallCommand = strDiff(*plan.InstallCommand, *state.InstallCommand)
	}
	if plan.BuildCommand != nil && state.BuildCommand != nil {
		input.BuildCommand = strDiff(*plan.BuildCommand, *state.BuildCommand)
	}
	if plan.StartCommand != nil && state.StartCommand != nil {
		input.StartCommand = strDiff(*plan.StartCommand, *state.StartCommand)
	}
	if plan.DockerfileLocation != nil && state.DockerfileLocation != nil {
		input.DockerfileLocation = strDiff(*plan.DockerfileLocation, *state.DockerfileLocation)
	}
	return input
}

// addExtendedUpdateFields adds extended fields to an UpdateApplicationInput,
// only including fields that differ between plan and state.
func addExtendedUpdateFields(plan, state commonAppFields, input *client.UpdateApplicationInput) {
	strDiff := flex.StringIfChanged
	boolDiff := flex.BoolIfChanged
	// Build/deploy
	input.BaseDirectory = strDiff(*plan.BaseDirectory, *state.BaseDirectory)
	input.PublishDirectory = strDiff(*plan.PublishDirectory, *state.PublishDirectory)
	input.DockerRegistryImageTag = strDiff(*plan.DockerRegistryImageTag, *state.DockerRegistryImageTag)
	input.DockerComposeDomains = dockerComposeDomainsIfChanged(*plan.DockerComposeDomains, *state.DockerComposeDomains)
	input.GitCommitSha = strDiff(*plan.GitCommitSha, *state.GitCommitSha)
	input.WatchPaths = strDiff(*plan.WatchPaths, *state.WatchPaths)
	// preview_url_template is not in Coolify v4's update $allowedFields.
	// Container/Network
	input.CustomDockerRunOptions = strDiff(*plan.CustomDockerRunOptions, *state.CustomDockerRunOptions)
	input.CustomLabels = strDiff(*plan.CustomLabels, *state.CustomLabels)
	flex.EncodeBase64Ptr(&input.CustomLabels)
	input.CustomNetworkAliases = strDiff(*plan.CustomNetworkAliases, *state.CustomNetworkAliases)
	input.CustomNginxConfiguration = strDiff(*plan.CustomNginxConfiguration, *state.CustomNginxConfiguration)
	flex.EncodeBase64Ptr(&input.CustomNginxConfiguration)
	input.PortsMappings = strDiff(*plan.PortsMappings, *state.PortsMappings)
	// Redirect & static
	input.Redirect = strDiff(*plan.Redirect, *state.Redirect)
	input.StaticImage = strDiff(*plan.StaticImage, *state.StaticImage)
	input.IsStatic = boolDiff(*plan.IsStatic, *state.IsStatic)
	input.IsSPA = boolDiff(*plan.IsSPA, *state.IsSPA)
	// Security & Auth
	input.IsForceHTTPSEnabled = boolDiff(*plan.IsForceHTTPSEnabled, *state.IsForceHTTPSEnabled)
	input.IsHTTPBasicAuthEnabled = boolDiff(*plan.IsHTTPBasicAuthEnabled, *state.IsHTTPBasicAuthEnabled)
	input.HTTPBasicAuthUsername = strDiff(*plan.HTTPBasicAuthUsername, *state.HTTPBasicAuthUsername)
	input.HTTPBasicAuthPassword = strDiff(*plan.HTTPBasicAuthPassword, *state.HTTPBasicAuthPassword)
	// Deployment commands
	input.PreDeploymentCommand = strDiff(*plan.PreDeploymentCommand, *state.PreDeploymentCommand)
	input.PreDeploymentCommandContainer = strDiff(*plan.PreDeploymentCommandContainer, *state.PreDeploymentCommandContainer)
	input.PostDeploymentCommand = strDiff(*plan.PostDeploymentCommand, *state.PostDeploymentCommand)
	input.PostDeploymentCommandContainer = strDiff(*plan.PostDeploymentCommandContainer, *state.PostDeploymentCommandContainer)
	// Webhook secrets
	input.ManualWebhookSecretBitbucket = strDiff(*plan.ManualWebhookSecretBitbucket, *state.ManualWebhookSecretBitbucket)
	input.ManualWebhookSecretGitea = strDiff(*plan.ManualWebhookSecretGitea, *state.ManualWebhookSecretGitea)
	input.ManualWebhookSecretGitHub = strDiff(*plan.ManualWebhookSecretGitHub, *state.ManualWebhookSecretGitHub)
	input.ManualWebhookSecretGitLab = strDiff(*plan.ManualWebhookSecretGitLab, *state.ManualWebhookSecretGitLab)
	// Other settings
	input.ConnectToDockerNetwork = boolDiff(*plan.ConnectToDockerNetwork, *state.ConnectToDockerNetwork)
	input.IsContainerLabelEscapeEnabled = boolDiff(*plan.IsContainerLabelEscapeEnabled, *state.IsContainerLabelEscapeEnabled)
	input.IsPreserveRepositoryEnabled = boolDiff(*plan.IsPreserveRepositoryEnabled, *state.IsPreserveRepositoryEnabled)
	input.UseBuildServer = boolDiff(*plan.UseBuildServer, *state.UseBuildServer)
	if plan.IsPreviewDeploymentsEnabled != nil && state.IsPreviewDeploymentsEnabled != nil {
		input.IsPreviewDeploymentsEnabled = boolDiff(*plan.IsPreviewDeploymentsEnabled, *state.IsPreviewDeploymentsEnabled)
	}
	if plan.UseBuildSecrets != nil && state.UseBuildSecrets != nil {
		input.UseBuildSecrets = boolDiff(*plan.UseBuildSecrets, *state.UseBuildSecrets)
	}
	if plan.StopGracePeriod != nil && state.StopGracePeriod != nil {
		input.StopGracePeriod = flex.Int64IfChanged(*plan.StopGracePeriod, *state.StopGracePeriod)
	}
	addApplicationSettingUpdateFields(plan, state, input)
	// Nil-safe resource-specific fields
	if plan.ForceDomainOverride != nil && state.ForceDomainOverride != nil {
		input.ForceDomainOverride = boolDiff(*plan.ForceDomainOverride, *state.ForceDomainOverride)
	}
	if plan.Dockerfile != nil && state.Dockerfile != nil {
		input.Dockerfile = strDiff(*plan.Dockerfile, *state.Dockerfile)
	}
	if plan.DockerfileTargetBuild != nil && state.DockerfileTargetBuild != nil {
		input.DockerfileTargetBuild = strDiff(*plan.DockerfileTargetBuild, *state.DockerfileTargetBuild)
	}
	if plan.DockerComposeLocation != nil && state.DockerComposeLocation != nil {
		input.DockerComposeLocation = strDiff(*plan.DockerComposeLocation, *state.DockerComposeLocation)
	}
	if plan.DockerComposeCustomBuildCommand != nil && state.DockerComposeCustomBuildCommand != nil {
		input.DockerComposeCustomBuildCommand = strDiff(*plan.DockerComposeCustomBuildCommand, *state.DockerComposeCustomBuildCommand)
	}
	if plan.DockerComposeCustomStartCommand != nil && state.DockerComposeCustomStartCommand != nil {
		input.DockerComposeCustomStartCommand = strDiff(*plan.DockerComposeCustomStartCommand, *state.DockerComposeCustomStartCommand)
	}
	if plan.MaxRestartCount != nil && state.MaxRestartCount != nil {
		input.MaxRestartCount = flex.Int64IfChanged(*plan.MaxRestartCount, *state.MaxRestartCount)
	}
}

// hasNonDefaultAppExtendedFields returns true if any field that the Create POST
// does not accept is configured with a non-default value, requiring a post-create
// PATCH to converge in a single apply.
func hasNonDefaultAppExtendedFields(f commonAppFields) bool {
	// Resource limits
	return flex.StringPtrNonDefault(f.LimitsMemory, "0") ||
		flex.StringPtrNonDefault(f.LimitsMemorySwap, "0") ||
		flex.StringPtrNonDefault(f.LimitsMemoryReservation, "0") ||
		flex.StringPtrNonDefault(f.LimitsCPUs, "0") ||
		flex.StringPtrNonDefault(f.LimitsCPUSet, "") ||
		flex.Int64PtrNonDefault(f.LimitsMemorySwappiness, 60) ||
		flex.Int64PtrNonDefault(f.LimitsCPUShares, 1024) ||
		// Health checks
		flex.BoolPtrNonDefault(f.HealthCheckEnabled, false) ||
		flex.StringPtrNonDefault(f.HealthCheckPath, "/") ||
		flex.StringPtrNonDefault(f.HealthCheckPort, "") ||
		flex.Int64PtrNonDefault(f.HealthCheckInterval, 5) ||
		flex.Int64PtrNonDefault(f.HealthCheckTimeout, 5) ||
		flex.Int64PtrNonDefault(f.HealthCheckRetries, 10) ||
		flex.Int64PtrNonDefault(f.HealthCheckStartPeriod, 5) ||
		flex.StringPtrNonDefault(f.HealthCheckCommand, "") ||
		flex.StringPtrNonDefault(f.HealthCheckHost, defaultHealthCheckHost) ||
		flex.StringPtrNonDefault(f.HealthCheckMethod, defaultHealthCheckMeth) ||
		flex.StringPtrNonDefault(f.HealthCheckResponseText, "") ||
		flex.Int64PtrNonDefault(f.HealthCheckReturnCode, defaultHealthCheckCode) ||
		flex.StringPtrNonDefault(f.HealthCheckScheme, defaultHealthCheckSchm) ||
		flex.StringPtrNonDefault(f.HealthCheckType, defaultHealthCheckType) ||
		// Auto-deploy
		flex.BoolPtrNonDefault(f.IsAutoDeployEnabled, true) ||
		// Build/deploy
		flex.StringPtrNonDefault(f.BaseDirectory, "") ||
		flex.StringPtrNonDefault(f.PublishDirectory, "") ||
		flex.StringPtrNonDefault(f.DockerRegistryImageTag, "") ||
		flex.StringPtrNonDefault(f.DockerComposeDomains, "") ||
		flex.StringPtrNonDefault(f.GitCommitSha, "") ||
		flex.StringPtrNonDefault(f.WatchPaths, "") ||
		// Container/Network
		flex.StringPtrNonDefault(f.CustomDockerRunOptions, "") ||
		flex.StringPtrNonDefault(f.CustomLabels, "") ||
		flex.StringPtrNonDefault(f.CustomNetworkAliases, "") ||
		flex.StringPtrNonDefault(f.CustomNginxConfiguration, "") ||
		flex.StringPtrNonDefault(f.PortsMappings, "") ||
		// Auth
		flex.BoolPtrNonDefault(f.IsHTTPBasicAuthEnabled, false) ||
		flex.StringPtrNonDefault(f.HTTPBasicAuthUsername, "") ||
		flex.StringPtrNonDefault(f.HTTPBasicAuthPassword, "") ||
		// Deployment commands
		flex.StringPtrNonDefault(f.PreDeploymentCommand, "") ||
		flex.StringPtrNonDefault(f.PreDeploymentCommandContainer, "") ||
		flex.StringPtrNonDefault(f.PostDeploymentCommand, "") ||
		flex.StringPtrNonDefault(f.PostDeploymentCommandContainer, "") ||
		// Webhook secrets (create POST omits these; Coolify auto-gens if unset) (#575)
		flex.StringPtrNonDefault(f.ManualWebhookSecretBitbucket, "") ||
		flex.StringPtrNonDefault(f.ManualWebhookSecretGitea, "") ||
		flex.StringPtrNonDefault(f.ManualWebhookSecretGitHub, "") ||
		flex.StringPtrNonDefault(f.ManualWebhookSecretGitLab, "") ||
		// Bool overrides
		flex.BoolPtrNonDefault(f.ConnectToDockerNetwork, false) ||
		flex.BoolPtrNonDefault(f.IsForceHTTPSEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsStatic, false) ||
		flex.BoolPtrNonDefault(f.IsSPA, false) ||
		flex.BoolPtrNonDefault(f.IsContainerLabelEscapeEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsPreserveRepositoryEnabled, false) ||
		flex.BoolPtrNonDefault(f.UseBuildServer, false) ||
		flex.BoolPtrNonDefault(f.IsPreviewDeploymentsEnabled, false) ||
		flex.BoolPtrNonDefault(f.UseBuildSecrets, false) ||
		(f.StopGracePeriod != nil && !f.StopGracePeriod.IsNull() && !f.StopGracePeriod.IsUnknown()) ||
		flex.BoolPtrNonDefault(f.IsGitSubmodulesEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsGitLfsEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsGitShallowCloneEnabled, true) ||
		flex.BoolPtrNonDefault(f.DisableBuildCache, false) ||
		flex.BoolPtrNonDefault(f.InjectBuildArgsToDockerfile, true) ||
		flex.BoolPtrNonDefault(f.IncludeSourceCommitInBuild, false) ||
		flex.BoolPtrNonDefault(f.IsEnvSortingEnabled, false) ||
		flex.BoolPtrNonDefault(f.IsPrDeploymentsPublicEnabled, false) ||
		(f.DockerImagesToKeep != nil && !f.DockerImagesToKeep.IsNull() && !f.DockerImagesToKeep.IsUnknown() && f.DockerImagesToKeep.ValueInt64() != 2) ||
		flex.BoolPtrNonDefault(f.IsGzipEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsStripprefixEnabled, true) ||
		flex.BoolPtrNonDefault(f.IsRawComposeDeploymentEnabled, false) ||
		flex.BoolPtrNonDefault(f.IsLogDrainEnabled, false) ||
		flex.BoolPtrNonDefault(f.IsGpuEnabled, false) ||
		flex.StringPtrNonDefault(f.GpuDriver, "nvidia") ||
		flex.StringPtrNonDefault(f.GpuCount, "") ||
		flex.StringPtrNonDefault(f.GpuDeviceIds, "") ||
		flex.StringPtrNonDefault(f.GpuOptions, "") ||
		flex.BoolPtrNonDefault(f.IsConsistentContainerNameEnabled, false) ||
		flex.StringPtrNonDefault(f.CustomInternalName, "") ||
		listPtrConfigured(f.NoindexDomains) ||
		flex.BoolPtrNonDefault(f.ForceDomainOverride, false) ||
		// String overrides
		flex.StringPtrNonDefault(f.Redirect, defaultRedirect) ||
		flex.StringPtrNonDefault(f.StaticImage, defaultStaticImage) ||
		flex.Int64PtrNonDefault(f.MaxRestartCount, 10)
}

// listPtrConfigured reports whether a List pointer is set (non-null, non-unknown).
// Empty lists count as configured so post-create can clear noindex_domains.
func listPtrConfigured(v *types.List) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown()
}

// buildPostCreatePatch builds an UpdateApplicationInput from the plan's extended
// fields, including only fields that are configured (non-null, non-unknown).
func buildPostCreatePatch(f commonAppFields) client.UpdateApplicationInput {
	var input client.UpdateApplicationInput
	safeStr := func(v *types.String) types.String {
		if v == nil {
			return types.StringNull()
		}
		return *v
	}
	safeInt := func(v *types.Int64) types.Int64 {
		if v == nil {
			return types.Int64Null()
		}
		return *v
	}
	safeBool := func(v *types.Bool) types.Bool {
		if v == nil {
			return types.BoolNull()
		}
		return *v
	}
	// Resource limits
	flex.SetStrPtr(&input.LimitsMemory, safeStr(f.LimitsMemory))
	flex.SetStrPtr(&input.LimitsMemorySwap, safeStr(f.LimitsMemorySwap))
	flex.SetInt64Ptr(&input.LimitsMemorySwappiness, safeInt(f.LimitsMemorySwappiness))
	flex.SetStrPtr(&input.LimitsMemoryReservation, safeStr(f.LimitsMemoryReservation))
	flex.SetStrPtr(&input.LimitsCPUs, safeStr(f.LimitsCPUs))
	flex.SetStrPtr(&input.LimitsCPUSet, safeStr(f.LimitsCPUSet))
	flex.SetInt64Ptr(&input.LimitsCPUShares, safeInt(f.LimitsCPUShares))
	// Health checks
	flex.SetBoolPtr(&input.HealthCheckEnabled, safeBool(f.HealthCheckEnabled))
	flex.SetStrPtr(&input.HealthCheckPath, safeStr(f.HealthCheckPath))
	flex.SetStrPtr(&input.HealthCheckPort, safeStr(f.HealthCheckPort))
	flex.SetInt64Ptr(&input.HealthCheckInterval, safeInt(f.HealthCheckInterval))
	flex.SetInt64Ptr(&input.HealthCheckTimeout, safeInt(f.HealthCheckTimeout))
	flex.SetInt64Ptr(&input.HealthCheckRetries, safeInt(f.HealthCheckRetries))
	flex.SetInt64Ptr(&input.HealthCheckStartPeriod, safeInt(f.HealthCheckStartPeriod))
	flex.SetStrPtr(&input.HealthCheckCommand, safeStr(f.HealthCheckCommand))
	flex.SetStrPtr(&input.HealthCheckHost, safeStr(f.HealthCheckHost))
	flex.SetStrPtr(&input.HealthCheckMethod, safeStr(f.HealthCheckMethod))
	flex.SetStrPtr(&input.HealthCheckResponseText, safeStr(f.HealthCheckResponseText))
	flex.SetInt64Ptr(&input.HealthCheckReturnCode, safeInt(f.HealthCheckReturnCode))
	flex.SetStrPtr(&input.HealthCheckScheme, safeStr(f.HealthCheckScheme))
	flex.SetStrPtr(&input.HealthCheckType, safeStr(f.HealthCheckType))
	// Auto-deploy
	flex.SetBoolPtr(&input.IsAutoDeployEnabled, safeBool(f.IsAutoDeployEnabled))
	// Build/deploy
	flex.SetStrPtr(&input.BaseDirectory, safeStr(f.BaseDirectory))
	flex.SetStrPtr(&input.PublishDirectory, safeStr(f.PublishDirectory))
	flex.SetStrPtr(&input.DockerRegistryImageTag, safeStr(f.DockerRegistryImageTag))
	if f.DockerComposeDomains != nil && !f.DockerComposeDomains.IsNull() && !f.DockerComposeDomains.IsUnknown() {
		input.DockerComposeDomains = wireDockerComposeDomains(f.DockerComposeDomains.ValueString())
	}
	flex.SetStrPtr(&input.GitCommitSha, safeStr(f.GitCommitSha))
	flex.SetStrPtr(&input.WatchPaths, safeStr(f.WatchPaths))
	// Container/Network
	flex.SetStrPtr(&input.CustomDockerRunOptions, safeStr(f.CustomDockerRunOptions))
	flex.SetStrPtr(&input.CustomLabels, safeStr(f.CustomLabels))
	flex.EncodeBase64Ptr(&input.CustomLabels)
	flex.SetStrPtr(&input.CustomNetworkAliases, safeStr(f.CustomNetworkAliases))
	flex.SetStrPtr(&input.CustomNginxConfiguration, safeStr(f.CustomNginxConfiguration))
	flex.EncodeBase64Ptr(&input.CustomNginxConfiguration)
	flex.SetStrPtr(&input.PortsMappings, safeStr(f.PortsMappings))
	// Redirect & static
	flex.SetStrPtr(&input.Redirect, safeStr(f.Redirect))
	flex.SetStrPtr(&input.StaticImage, safeStr(f.StaticImage))
	flex.SetBoolPtr(&input.IsStatic, safeBool(f.IsStatic))
	flex.SetBoolPtr(&input.IsSPA, safeBool(f.IsSPA))
	// Security & Auth
	flex.SetBoolPtr(&input.IsForceHTTPSEnabled, safeBool(f.IsForceHTTPSEnabled))
	flex.SetBoolPtr(&input.IsHTTPBasicAuthEnabled, safeBool(f.IsHTTPBasicAuthEnabled))
	flex.SetStrPtr(&input.HTTPBasicAuthUsername, safeStr(f.HTTPBasicAuthUsername))
	flex.SetStrPtr(&input.HTTPBasicAuthPassword, safeStr(f.HTTPBasicAuthPassword))
	// Deployment commands
	flex.SetStrPtr(&input.PreDeploymentCommand, safeStr(f.PreDeploymentCommand))
	flex.SetStrPtr(&input.PreDeploymentCommandContainer, safeStr(f.PreDeploymentCommandContainer))
	flex.SetStrPtr(&input.PostDeploymentCommand, safeStr(f.PostDeploymentCommand))
	flex.SetStrPtr(&input.PostDeploymentCommandContainer, safeStr(f.PostDeploymentCommandContainer))
	// Webhook secrets (not on create POST; set via post-create PATCH) (#575)
	flex.SetStrPtr(&input.ManualWebhookSecretBitbucket, safeStr(f.ManualWebhookSecretBitbucket))
	flex.SetStrPtr(&input.ManualWebhookSecretGitea, safeStr(f.ManualWebhookSecretGitea))
	flex.SetStrPtr(&input.ManualWebhookSecretGitHub, safeStr(f.ManualWebhookSecretGitHub))
	flex.SetStrPtr(&input.ManualWebhookSecretGitLab, safeStr(f.ManualWebhookSecretGitLab))
	// Other settings
	flex.SetBoolPtr(&input.ConnectToDockerNetwork, safeBool(f.ConnectToDockerNetwork))
	flex.SetBoolPtr(&input.IsContainerLabelEscapeEnabled, safeBool(f.IsContainerLabelEscapeEnabled))
	flex.SetBoolPtr(&input.IsPreserveRepositoryEnabled, safeBool(f.IsPreserveRepositoryEnabled))
	flex.SetBoolPtr(&input.UseBuildServer, safeBool(f.UseBuildServer))
	flex.SetBoolPtr(&input.IsPreviewDeploymentsEnabled, safeBool(f.IsPreviewDeploymentsEnabled))
	flex.SetBoolPtr(&input.UseBuildSecrets, safeBool(f.UseBuildSecrets))
	flex.SetInt64Ptr(&input.StopGracePeriod, safeInt(f.StopGracePeriod))
	setApplicationSettingPostCreate(&input, f, safeBool, safeInt)
	flex.SetInt64Ptr(&input.MaxRestartCount, safeInt(f.MaxRestartCount))
	flex.SetBoolPtr(&input.ForceDomainOverride, safeBool(f.ForceDomainOverride))
	return input
}

// postCreatePatchExtendedFields sends a PATCH after Create when the plan includes
// extended fields not accepted by the Create POST (resource limits, health checks,
// deployment commands, auth, custom docker options, etc.). Without this, those
// fields cause "Provider produced inconsistent result after apply".
func postCreatePatchExtendedFields(ctx context.Context, c *client.Client, uuid string, f commonAppFields, resp *resource.CreateResponse) {
	if !hasNonDefaultAppExtendedFields(f) {
		return
	}
	input := buildPostCreatePatch(f)
	// On Coolify below the settings gates the client strips version-gated write
	// fields, so a plan whose only extended fields are those would PATCH an
	// empty body. Skip it rather than spend a request that can change nothing.
	if !c.SupportsApplicationSettings() && input.HasOnlyApplicationSettings() {
		tflog.Debug(ctx, "skipping post-create patch: only Coolify>=4.2.0 write fields, unsupported on this Coolify version",
			map[string]interface{}{"uuid": uuid})
		return
	}
	if c.SupportsApplicationSettings() && !c.SupportsApplicationSettingsV43() && input.HasOnlyApplicationSettingsV43() {
		tflog.Debug(ctx, "skipping post-create patch: only Coolify>=4.3.0 write fields, unsupported on this Coolify version",
			map[string]interface{}{"uuid": uuid})
		return
	}
	tflog.Debug(ctx, "patching extended fields after create", map[string]interface{}{"uuid": uuid})
	if _, err := c.UpdateApplication(ctx, uuid, input); err != nil {
		hint := annotateDockerComposeDomainsError(err)
		converge := ""
		if hint == "" {
			// Re-apply alone does not help for docker_compose_domains empty-raw;
			// that path uses annotateDockerComposeDomainsError instead.
			converge = " Run terraform apply again to converge."
		}
		resp.Diagnostics.AddError("Error setting application extended fields",
			fmt.Sprintf("Application %s was created, but the post-create PATCH for extended fields failed: %s.%s%s",
				uuid, err, converge, hint))
	}
}

// setBoolDefault sets a bool pointer field from API value or schema default.
func setBoolDefault(dst *types.Bool, v *bool, def bool) {
	if dst == nil {
		return
	}
	if v != nil {
		*dst = types.BoolValue(*v)
	} else if dst.IsNull() || dst.IsUnknown() {
		*dst = types.BoolValue(def)
	}
}

// flattenRestartLimitFields maps GET-only restart-limit status plus the
// writable max_restart_count. Extracted so flattenExtendedDefaults stays
// under the gocognit limit.
func flattenRestartLimitFields(app *client.Application, f commonAppFields) {
	if f.MaxRestartCount != nil {
		*f.MaxRestartCount = flex.Int64PtrToFramework(app.MaxRestartCount)
	}
	setBoolOrNull(f.RestartLimitReached, app.RestartLimitReached)
	setBoolOrNull(f.ContainerPresent, app.ContainerPresent)
}

func setBoolOrNull(dst *types.Bool, v *bool) {
	if dst == nil {
		return
	}
	if v != nil {
		*dst = types.BoolValue(*v)
	} else {
		*dst = types.BoolNull()
	}
}

func flattenApplicationSettingFields(app *client.Application, f commonAppFields) {
	setBoolDefault(f.IsGitSubmodulesEnabled, app.IsGitSubmodulesEnabled, true)
	setBoolDefault(f.IsGitLfsEnabled, app.IsGitLfsEnabled, true)
	setBoolDefault(f.IsGitShallowCloneEnabled, app.IsGitShallowCloneEnabled, true)
	setBoolDefault(f.DisableBuildCache, app.DisableBuildCache, false)
	setBoolDefault(f.InjectBuildArgsToDockerfile, app.InjectBuildArgsToDockerfile, true)
	setBoolDefault(f.IncludeSourceCommitInBuild, app.IncludeSourceCommitInBuild, false)
	setBoolDefault(f.IsEnvSortingEnabled, app.IsEnvSortingEnabled, false)
	setBoolDefault(f.IsPrDeploymentsPublicEnabled, app.IsPrDeploymentsPublicEnabled, false)
	if f.DockerImagesToKeep != nil {
		if app.DockerImagesToKeep != nil {
			*f.DockerImagesToKeep = types.Int64Value(*app.DockerImagesToKeep)
		} else if f.DockerImagesToKeep.IsNull() || f.DockerImagesToKeep.IsUnknown() {
			*f.DockerImagesToKeep = types.Int64Value(2)
		}
	}
	setBoolDefault(f.IsGzipEnabled, app.IsGzipEnabled, true)
	setBoolDefault(f.IsStripprefixEnabled, app.IsStripprefixEnabled, true)
	setBoolDefault(f.IsRawComposeDeploymentEnabled, app.IsRawComposeDeploymentEnabled, false)
	// Coolify >= v4.3.0 fields (defaults applied only when null/unknown).
	setBoolDefault(f.IsLogDrainEnabled, app.IsLogDrainEnabled, false)
	setBoolDefault(f.IsGpuEnabled, app.IsGpuEnabled, false)
	if f.GpuDriver != nil {
		if app.GpuDriver != "" {
			*f.GpuDriver = types.StringValue(app.GpuDriver)
		} else if f.GpuDriver.IsNull() || f.GpuDriver.IsUnknown() {
			*f.GpuDriver = types.StringValue("nvidia")
		}
	}
	flex.SetStringSeedOrClear(f.GpuCount, app.GpuCount)
	flex.SetStringSeedOrClear(f.GpuDeviceIds, app.GpuDeviceIds)
	flex.SetStringSeedOrClear(f.GpuOptions, app.GpuOptions)
	setBoolDefault(f.IsConsistentContainerNameEnabled, app.IsConsistentContainerNameEnabled, false)
	flex.SetStringSeedOrClear(f.CustomInternalName, app.CustomInternalName)
	flattenNoindexDomains(app.NoindexDomains, f.NoindexDomains)
}

func flattenNoindexDomains(api []string, dst *types.List) {
	if dst == nil {
		return
	}
	if len(api) == 0 {
		if dst.IsNull() || dst.IsUnknown() {
			*dst = types.ListNull(types.StringType)
			return
		}
		// Configured (including empty list): keep current. Coolify < 4.3
		// omits noindex_domains on GET; writing [] would fight the write gate.
		return
	}
	if dst.IsNull() || dst.IsUnknown() {
		*dst = stringListValue(api)
		return
	}
	// Coolify stores a JSON array and may unique/normalize URLs. Keep the
	// configured list order (and original casing) when the set matches.
	if stringListEquivalent(stringListFromTypes(*dst), api) {
		return
	}
	*dst = stringListValue(api)
}

func stringListEquivalent(configured, api []string) bool {
	if len(configured) != len(api) {
		return false
	}
	counts := make(map[string]int, len(api))
	for _, s := range api {
		counts[strings.ToLower(s)]++
	}
	for _, s := range configured {
		k := strings.ToLower(s)
		n := counts[k]
		if n == 0 {
			return false
		}
		counts[k] = n - 1
	}
	return true
}

func stringListValue(items []string) types.List {
	elems := make([]attr.Value, len(items))
	for i, s := range items {
		elems[i] = types.StringValue(s)
	}
	return types.ListValueMust(types.StringType, elems)
}

func stringListFromTypes(v types.List) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	elems := v.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		sv, ok := e.(types.String)
		if !ok || sv.IsNull() || sv.IsUnknown() {
			continue
		}
		out = append(out, sv.ValueString())
	}
	return out
}

func addApplicationSettingUpdateFields(plan, state commonAppFields, input *client.UpdateApplicationInput) {
	setBoolDiff := func(dst **bool, p, s *types.Bool) {
		if p != nil && s != nil {
			*dst = flex.BoolIfChanged(*p, *s)
		}
	}
	setStrDiff := func(dst **string, p, s *types.String) {
		if p != nil && s != nil {
			*dst = flex.StringIfChanged(*p, *s)
		}
	}
	setBoolDiff(&input.IsGitSubmodulesEnabled, plan.IsGitSubmodulesEnabled, state.IsGitSubmodulesEnabled)
	setBoolDiff(&input.IsGitLfsEnabled, plan.IsGitLfsEnabled, state.IsGitLfsEnabled)
	setBoolDiff(&input.IsGitShallowCloneEnabled, plan.IsGitShallowCloneEnabled, state.IsGitShallowCloneEnabled)
	setBoolDiff(&input.DisableBuildCache, plan.DisableBuildCache, state.DisableBuildCache)
	setBoolDiff(&input.InjectBuildArgsToDockerfile, plan.InjectBuildArgsToDockerfile, state.InjectBuildArgsToDockerfile)
	setBoolDiff(&input.IncludeSourceCommitInBuild, plan.IncludeSourceCommitInBuild, state.IncludeSourceCommitInBuild)
	setBoolDiff(&input.IsEnvSortingEnabled, plan.IsEnvSortingEnabled, state.IsEnvSortingEnabled)
	setBoolDiff(&input.IsPrDeploymentsPublicEnabled, plan.IsPrDeploymentsPublicEnabled, state.IsPrDeploymentsPublicEnabled)
	setBoolDiff(&input.IsGzipEnabled, plan.IsGzipEnabled, state.IsGzipEnabled)
	setBoolDiff(&input.IsStripprefixEnabled, plan.IsStripprefixEnabled, state.IsStripprefixEnabled)
	setBoolDiff(&input.IsRawComposeDeploymentEnabled, plan.IsRawComposeDeploymentEnabled, state.IsRawComposeDeploymentEnabled)
	if plan.DockerImagesToKeep != nil && state.DockerImagesToKeep != nil {
		input.DockerImagesToKeep = flex.Int64IfChanged(*plan.DockerImagesToKeep, *state.DockerImagesToKeep)
	}
	setBoolDiff(&input.IsLogDrainEnabled, plan.IsLogDrainEnabled, state.IsLogDrainEnabled)
	setBoolDiff(&input.IsGpuEnabled, plan.IsGpuEnabled, state.IsGpuEnabled)
	setStrDiff(&input.GpuDriver, plan.GpuDriver, state.GpuDriver)
	setStrDiff(&input.GpuCount, plan.GpuCount, state.GpuCount)
	setStrDiff(&input.GpuDeviceIds, plan.GpuDeviceIds, state.GpuDeviceIds)
	setStrDiff(&input.GpuOptions, plan.GpuOptions, state.GpuOptions)
	setBoolDiff(&input.IsConsistentContainerNameEnabled, plan.IsConsistentContainerNameEnabled, state.IsConsistentContainerNameEnabled)
	setStrDiff(&input.CustomInternalName, plan.CustomInternalName, state.CustomInternalName)
	if plan.NoindexDomains != nil && state.NoindexDomains != nil && !plan.NoindexDomains.Equal(*state.NoindexDomains) {
		if plan.NoindexDomains.IsNull() {
			empty := []string{}
			input.NoindexDomains = &empty
		} else if !plan.NoindexDomains.IsUnknown() {
			list := stringListFromTypes(*plan.NoindexDomains)
			input.NoindexDomains = &list
		}
	}
}

func setApplicationSettingPostCreate(input *client.UpdateApplicationInput, f commonAppFields, safeBool func(*types.Bool) types.Bool, safeInt func(*types.Int64) types.Int64) {
	flex.SetBoolPtr(&input.IsGitSubmodulesEnabled, safeBool(f.IsGitSubmodulesEnabled))
	flex.SetBoolPtr(&input.IsGitLfsEnabled, safeBool(f.IsGitLfsEnabled))
	flex.SetBoolPtr(&input.IsGitShallowCloneEnabled, safeBool(f.IsGitShallowCloneEnabled))
	flex.SetBoolPtr(&input.DisableBuildCache, safeBool(f.DisableBuildCache))
	flex.SetBoolPtr(&input.InjectBuildArgsToDockerfile, safeBool(f.InjectBuildArgsToDockerfile))
	flex.SetBoolPtr(&input.IncludeSourceCommitInBuild, safeBool(f.IncludeSourceCommitInBuild))
	flex.SetBoolPtr(&input.IsEnvSortingEnabled, safeBool(f.IsEnvSortingEnabled))
	flex.SetBoolPtr(&input.IsPrDeploymentsPublicEnabled, safeBool(f.IsPrDeploymentsPublicEnabled))
	flex.SetInt64Ptr(&input.DockerImagesToKeep, safeInt(f.DockerImagesToKeep))
	flex.SetBoolPtr(&input.IsGzipEnabled, safeBool(f.IsGzipEnabled))
	flex.SetBoolPtr(&input.IsStripprefixEnabled, safeBool(f.IsStripprefixEnabled))
	flex.SetBoolPtr(&input.IsRawComposeDeploymentEnabled, safeBool(f.IsRawComposeDeploymentEnabled))
	flex.SetBoolPtr(&input.IsLogDrainEnabled, safeBool(f.IsLogDrainEnabled))
	flex.SetBoolPtr(&input.IsGpuEnabled, safeBool(f.IsGpuEnabled))
	if f.GpuDriver != nil {
		flex.SetStrPtr(&input.GpuDriver, *f.GpuDriver)
	}
	if f.GpuCount != nil {
		flex.SetStrPtr(&input.GpuCount, *f.GpuCount)
	}
	if f.GpuDeviceIds != nil {
		flex.SetStrPtr(&input.GpuDeviceIds, *f.GpuDeviceIds)
	}
	if f.GpuOptions != nil {
		flex.SetStrPtr(&input.GpuOptions, *f.GpuOptions)
	}
	flex.SetBoolPtr(&input.IsConsistentContainerNameEnabled, safeBool(f.IsConsistentContainerNameEnabled))
	if f.CustomInternalName != nil {
		flex.SetStrPtr(&input.CustomInternalName, *f.CustomInternalName)
	}
	if listPtrConfigured(f.NoindexDomains) {
		list := stringListFromTypes(*f.NoindexDomains)
		input.NoindexDomains = &list
	}
}
