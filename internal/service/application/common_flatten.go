package application

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
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
	*f.Domains = flex.StringToFramework(app.Domains)
	// Coolify does not return dockerfile_location on GET for most app types.
	// Preserve the user's configured value to avoid "inconsistent result after apply".
	// The value IS sent on Create/Update, just not returned on read-back.
	if f.DockerfileLocation != nil && app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	flex.SetStringOrClear(f.InstallCommand, app.InstallCommand)
	flex.SetStringOrClear(f.BuildCommand, app.BuildCommand)
	flex.SetStringOrClear(f.StartCommand, app.StartCommand)
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
	// NOT NULL fields with DB defaults — API always returns a value.
	flex.SetStringIfConfigured(f.BaseDirectory, app.BaseDirectory)
	flex.SetStringIfConfigured(f.GitCommitSha, app.GitCommitSha)
	// Sensitive field hidden by API without read:sensitive permission.
	flex.SetStringIfConfigured(f.CustomLabels, app.CustomLabels)
	// Nullable fields — use SetStringOrClear so drift is detected when
	// someone clears the field in the Coolify UI.
	flex.SetStringOrClear(f.PublishDirectory, app.PublishDirectory)
	flex.SetStringIfConfigured(f.Dockerfile, app.Dockerfile)
	flex.SetStringOrClear(f.DockerRegistryImageTag, app.DockerRegistryImageTag)
	flex.SetStringOrClear(f.DockerComposeDomains, app.DockerComposeDomains)
	flex.SetStringOrClear(f.WatchPaths, app.WatchPaths)
	flex.SetStringOrClear(f.CustomDockerRunOptions, app.CustomDockerRunOptions)
	flex.SetStringOrClear(f.CustomNetworkAliases, app.CustomNetworkAliases)
	flex.SetStringOrClear(f.CustomNginxConfiguration, app.CustomNginxConfiguration)
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
	setString(f.ManualWebhookSecretBitbucket, flex.StringToFramework(app.ManualWebhookSecretBitbucket))
	setString(f.ManualWebhookSecretGitea, flex.StringToFramework(app.ManualWebhookSecretGitea))
	setString(f.ManualWebhookSecretGitHub, flex.StringToFramework(app.ManualWebhookSecretGitHub))
	setString(f.ManualWebhookSecretGitLab, flex.StringToFramework(app.ManualWebhookSecretGitLab))
	// Computed+Default bool fields (always set from API)
	setBoolDefault := func(dst *types.Bool, v *bool, def bool) {
		if dst == nil {
			return
		}
		if v != nil {
			*dst = types.BoolValue(*v)
			return
		}
		*dst = types.BoolValue(def)
	}
	setBoolDefault(f.ConnectToDockerNetwork, app.ConnectToDockerNetwork, false)
	setBoolDefault(f.IsHTTPBasicAuthEnabled, app.IsHTTPBasicAuthEnabled, false)
	setBoolDefault(f.IsStatic, app.IsStatic, false)
	setBoolDefault(f.IsSPA, app.IsSPA, false)
	setBoolDefault(f.IsForceHTTPSEnabled, app.IsForceHTTPSEnabled, true)
	setBoolDefault(f.IsContainerLabelEscapeEnabled, app.IsContainerLabelEscapeEnabled, true)
	setBoolDefault(f.IsPreserveRepositoryEnabled, app.IsPreserveRepositoryEnabled, false)
	setBoolDefault(f.UseBuildServer, app.UseBuildServer, false)
	// instant_deploy is create-only and never returned by the API.
	// Preserve state value when set; default to false otherwise (import).
	if f.InstantDeploy != nil && (f.InstantDeploy.IsNull() || f.InstantDeploy.IsUnknown()) {
		*f.InstantDeploy = types.BoolValue(false)
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
	input.DockerComposeDomains = strDiff(*plan.DockerComposeDomains, *state.DockerComposeDomains)
	input.GitCommitSha = strDiff(*plan.GitCommitSha, *state.GitCommitSha)
	input.WatchPaths = strDiff(*plan.WatchPaths, *state.WatchPaths)
	// preview_url_template is not in Coolify v4's update $allowedFields.
	// Container/Network
	input.CustomDockerRunOptions = strDiff(*plan.CustomDockerRunOptions, *state.CustomDockerRunOptions)
	input.CustomLabels = strDiff(*plan.CustomLabels, *state.CustomLabels)
	input.CustomNetworkAliases = strDiff(*plan.CustomNetworkAliases, *state.CustomNetworkAliases)
	input.CustomNginxConfiguration = strDiff(*plan.CustomNginxConfiguration, *state.CustomNginxConfiguration)
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
}

// hasNonDefaultAppExtendedFields returns true if any field that the Create POST
// does not accept is configured with a non-default value, requiring a post-create
// PATCH to converge in a single apply.
func hasNonDefaultAppExtendedFields(f commonAppFields) bool {
	strNonDefault := func(v *types.String, dflt string) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueString() != dflt
	}
	strSet := func(v *types.String) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
	}
	intNonDefault := func(v *types.Int64, dflt int64) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != dflt
	}
	boolNonDefault := func(v *types.Bool, dflt bool) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueBool() != dflt
	}
	// Resource limits
	return strNonDefault(f.LimitsMemory, "0") ||
		strNonDefault(f.LimitsMemorySwap, "0") ||
		strNonDefault(f.LimitsMemoryReservation, "0") ||
		strNonDefault(f.LimitsCPUs, "0") ||
		strSet(f.LimitsCPUSet) ||
		intNonDefault(f.LimitsMemorySwappiness, 60) ||
		intNonDefault(f.LimitsCPUShares, 1024) ||
		// Health checks
		boolNonDefault(f.HealthCheckEnabled, false) ||
		strNonDefault(f.HealthCheckPath, "/") ||
		strSet(f.HealthCheckPort) ||
		intNonDefault(f.HealthCheckInterval, 5) ||
		intNonDefault(f.HealthCheckTimeout, 5) ||
		intNonDefault(f.HealthCheckRetries, 10) ||
		intNonDefault(f.HealthCheckStartPeriod, 5) ||
		strSet(f.HealthCheckCommand) ||
		strNonDefault(f.HealthCheckHost, defaultHealthCheckHost) ||
		strNonDefault(f.HealthCheckMethod, defaultHealthCheckMeth) ||
		strSet(f.HealthCheckResponseText) ||
		intNonDefault(f.HealthCheckReturnCode, defaultHealthCheckCode) ||
		strNonDefault(f.HealthCheckScheme, defaultHealthCheckSchm) ||
		strNonDefault(f.HealthCheckType, defaultHealthCheckType) ||
		// Auto-deploy
		boolNonDefault(f.IsAutoDeployEnabled, true) ||
		// Build/deploy
		strSet(f.BaseDirectory) ||
		strSet(f.PublishDirectory) ||
		strSet(f.DockerRegistryImageTag) ||
		strSet(f.DockerComposeDomains) ||
		strSet(f.GitCommitSha) ||
		strSet(f.WatchPaths) ||
		// Container/Network
		strSet(f.CustomDockerRunOptions) ||
		strSet(f.CustomLabels) ||
		strSet(f.CustomNetworkAliases) ||
		strSet(f.CustomNginxConfiguration) ||
		strSet(f.PortsMappings) ||
		// Auth
		boolNonDefault(f.IsHTTPBasicAuthEnabled, false) ||
		strSet(f.HTTPBasicAuthUsername) ||
		strSet(f.HTTPBasicAuthPassword) ||
		// Deployment commands
		strSet(f.PreDeploymentCommand) ||
		strSet(f.PreDeploymentCommandContainer) ||
		strSet(f.PostDeploymentCommand) ||
		strSet(f.PostDeploymentCommandContainer) ||
		// Bool overrides
		boolNonDefault(f.ConnectToDockerNetwork, false) ||
		boolNonDefault(f.IsForceHTTPSEnabled, true) ||
		boolNonDefault(f.IsStatic, false) ||
		boolNonDefault(f.IsSPA, false) ||
		boolNonDefault(f.IsContainerLabelEscapeEnabled, true) ||
		boolNonDefault(f.IsPreserveRepositoryEnabled, false) ||
		boolNonDefault(f.UseBuildServer, false) ||
		boolNonDefault(f.ForceDomainOverride, false) ||
		// String overrides
		strNonDefault(f.Redirect, defaultRedirect) ||
		strNonDefault(f.StaticImage, defaultStaticImage)
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
	flex.SetStrPtr(&input.DockerComposeDomains, safeStr(f.DockerComposeDomains))
	flex.SetStrPtr(&input.GitCommitSha, safeStr(f.GitCommitSha))
	flex.SetStrPtr(&input.WatchPaths, safeStr(f.WatchPaths))
	// Container/Network
	flex.SetStrPtr(&input.CustomDockerRunOptions, safeStr(f.CustomDockerRunOptions))
	flex.SetStrPtr(&input.CustomLabels, safeStr(f.CustomLabels))
	flex.SetStrPtr(&input.CustomNetworkAliases, safeStr(f.CustomNetworkAliases))
	flex.SetStrPtr(&input.CustomNginxConfiguration, safeStr(f.CustomNginxConfiguration))
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
	// Other settings
	flex.SetBoolPtr(&input.ConnectToDockerNetwork, safeBool(f.ConnectToDockerNetwork))
	flex.SetBoolPtr(&input.IsContainerLabelEscapeEnabled, safeBool(f.IsContainerLabelEscapeEnabled))
	flex.SetBoolPtr(&input.IsPreserveRepositoryEnabled, safeBool(f.IsPreserveRepositoryEnabled))
	flex.SetBoolPtr(&input.UseBuildServer, safeBool(f.UseBuildServer))
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
	tflog.Debug(ctx, "patching extended fields after create", map[string]interface{}{"uuid": uuid})
	input := buildPostCreatePatch(f)
	if _, err := c.UpdateApplication(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error setting application extended fields",
			fmt.Sprintf("Application %s was created, but the post-create PATCH for extended fields failed: %s. "+
				"Run terraform apply again to converge.", uuid, err))
	}
}
