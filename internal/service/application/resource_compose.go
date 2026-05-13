package application

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dockerComposeApplicationResource{}
	_ resource.ResourceWithConfigure   = &dockerComposeApplicationResource{}
	_ resource.ResourceWithImportState = &dockerComposeApplicationResource{}
)

// dockerComposeApplicationResource manages a Coolify application deployed from a Docker Compose file.
type dockerComposeApplicationResource struct {
	client *client.Client
}

//nolint:dupl // model structs are intentionally similar across resource types
type dockerComposeApplicationResourceModel struct {
	UUID                            types.String   `tfsdk:"uuid"`
	Name                            types.String   `tfsdk:"name"`
	Description                     types.String   `tfsdk:"description"`
	ProjectUUID                     types.String   `tfsdk:"project_uuid"`
	ServerUUID                      types.String   `tfsdk:"server_uuid"`
	EnvironmentName                 types.String   `tfsdk:"environment_name"`
	DockerComposeRaw                types.String   `tfsdk:"docker_compose_raw"`
	FQDN                            types.String   `tfsdk:"fqdn"`
	Status                          types.String   `tfsdk:"status"`
	LimitsMemory                    types.String   `tfsdk:"limits_memory"`
	LimitsMemorySwap                types.String   `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness          types.Int64    `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation         types.String   `tfsdk:"limits_memory_reservation"`
	LimitsCPUs                      types.String   `tfsdk:"limits_cpus"`
	LimitsCPUSet                    types.String   `tfsdk:"limits_cpuset"`
	LimitsCPUShares                 types.Int64    `tfsdk:"limits_cpu_shares"`
	HealthCheckEnabled              types.Bool     `tfsdk:"health_check_enabled"`
	HealthCheckPath                 types.String   `tfsdk:"health_check_path"`
	HealthCheckPort                 types.String   `tfsdk:"health_check_port"`
	HealthCheckInterval             types.Int64    `tfsdk:"health_check_interval"`
	HealthCheckTimeout              types.Int64    `tfsdk:"health_check_timeout"`
	HealthCheckRetries              types.Int64    `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod          types.Int64    `tfsdk:"health_check_start_period"`
	HealthCheckCommand              types.String   `tfsdk:"health_check_command"`
	HealthCheckHost                 types.String   `tfsdk:"health_check_host"`
	HealthCheckMethod               types.String   `tfsdk:"health_check_method"`
	HealthCheckResponseText         types.String   `tfsdk:"health_check_response_text"`
	HealthCheckReturnCode           types.Int64    `tfsdk:"health_check_return_code"`
	HealthCheckScheme               types.String   `tfsdk:"health_check_scheme"`
	HealthCheckType                 types.String   `tfsdk:"health_check_type"`
	IsAutoDeployEnabled             types.Bool     `tfsdk:"is_auto_deploy_enabled"`
	BaseDirectory                   types.String   `tfsdk:"base_directory"`
	Dockerfile                      types.String   `tfsdk:"dockerfile"`
	DockerRegistryImageTag          types.String   `tfsdk:"docker_registry_image_tag"`
	DockerComposeDomains            types.String   `tfsdk:"docker_compose_domains"`
	DockerComposeLocation           types.String   `tfsdk:"docker_compose_location"`
	DockerComposeCustomBuildCommand types.String   `tfsdk:"docker_compose_custom_build_command"`
	DockerComposeCustomStartCommand types.String   `tfsdk:"docker_compose_custom_start_command"`
	GitCommitSha                    types.String   `tfsdk:"git_commit_sha"`
	PublishDirectory                types.String   `tfsdk:"publish_directory"`
	WatchPaths                      types.String   `tfsdk:"watch_paths"`
	PreviewURLTemplate              types.String   `tfsdk:"preview_url_template"`
	CustomDockerRunOptions          types.String   `tfsdk:"custom_docker_run_options"`
	CustomLabels                    types.String   `tfsdk:"custom_labels"`
	CustomNetworkAliases            types.String   `tfsdk:"custom_network_aliases"`
	CustomNginxConfiguration        types.String   `tfsdk:"custom_nginx_configuration"`
	PortsMappings                   types.String   `tfsdk:"ports_mappings"`
	ConnectToDockerNetwork          types.Bool     `tfsdk:"connect_to_docker_network"`
	Redirect                        types.String   `tfsdk:"redirect"`
	StaticImage                     types.String   `tfsdk:"static_image"`
	IsStatic                        types.Bool     `tfsdk:"is_static"`
	IsSPA                           types.Bool     `tfsdk:"is_spa"`
	IsForceHTTPSEnabled             types.Bool     `tfsdk:"is_force_https_enabled"`
	IsHTTPBasicAuthEnabled          types.Bool     `tfsdk:"is_http_basic_auth_enabled"`
	HTTPBasicAuthUsername           types.String   `tfsdk:"http_basic_auth_username"`
	HTTPBasicAuthPassword           types.String   `tfsdk:"http_basic_auth_password"`
	PreDeploymentCommand            types.String   `tfsdk:"pre_deployment_command"`
	PreDeploymentCommandContainer   types.String   `tfsdk:"pre_deployment_command_container"`
	PostDeploymentCommand           types.String   `tfsdk:"post_deployment_command"`
	PostDeploymentCommandContainer  types.String   `tfsdk:"post_deployment_command_container"`
	ManualWebhookSecretBitbucket    types.String   `tfsdk:"manual_webhook_secret_bitbucket"`
	ManualWebhookSecretGitea        types.String   `tfsdk:"manual_webhook_secret_gitea"`
	ManualWebhookSecretGitHub       types.String   `tfsdk:"manual_webhook_secret_github"`
	ManualWebhookSecretGitLab       types.String   `tfsdk:"manual_webhook_secret_gitlab"`
	ForceDomainOverride             types.Bool     `tfsdk:"force_domain_override"`
	IsContainerLabelEscapeEnabled   types.Bool     `tfsdk:"is_container_label_escape_enabled"`
	IsPreserveRepositoryEnabled     types.Bool     `tfsdk:"is_preserve_repository_enabled"`
	UseBuildServer                  types.Bool     `tfsdk:"use_build_server"`
	Timeouts                        timeouts.Value `tfsdk:"timeouts"`
}

// NewDockerComposeResource returns a new dockerComposeApplicationResource instance.
func NewDockerComposeResource() resource.Resource {
	return &dockerComposeApplicationResource{}
}

func (r *dockerComposeApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_compose_application"
}

func (r *dockerComposeApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Docker Compose file.",
		Attributes: CommonAppAttrs(ctx, map[string]schema.Attribute{
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "The Docker Compose YAML content, **base64-encoded**. Use `base64encode(<<-YAML ... YAML)` in your configuration. Despite the field name, Coolify's API requires base64 encoding.",
				Required:            true,
			},
			"docker_compose_location": schema.StringAttribute{
				MarkdownDescription: "The location of the Docker Compose file relative to the repository root.",
				Optional:            true,
			},
			"docker_compose_custom_build_command": schema.StringAttribute{
				MarkdownDescription: "Custom build command for Docker Compose.",
				Optional:            true,
			},
			"docker_compose_custom_start_command": schema.StringAttribute{
				MarkdownDescription: "Custom start command for Docker Compose.",
				Optional:            true,
			},
		}),
	}
}

func (r *dockerComposeApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *dockerComposeApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_docker_compose_application"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateDockerComposeAppInput{
		ProjectUUID:      plan.ProjectUUID.ValueString(),
		ServerUUID:       plan.ServerUUID.ValueString(),
		DockerComposeRaw: plan.DockerComposeRaw.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)

	created, err := r.client.CreateDockerComposeApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating docker compose application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := readBackAfterCreate(ctx, r.client, created.UUID, resp)
	if app == nil {
		return
	}

	flattenDockerComposeApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerComposeApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_docker_compose_application", "uuid": state.UUID.ValueString()})

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenDockerComposeApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerComposeApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_docker_compose_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common())
	input.DockerComposeRaw = flex.StringValueOrNull(plan.DockerComposeRaw)

	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenDockerComposeApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerComposeApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_docker_compose_application", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *dockerComposeApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	setImportDefaults(ctx, resp)
}

//nolint:dupl // .common() methods differ by receiver type
func (m *dockerComposeApplicationResourceModel) common() commonAppFields {
	return commonAppFields{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		FQDN:   &m.FQDN,
		Status: &m.Status, ProjectUUID: &m.ProjectUUID, ServerUUID: &m.ServerUUID,
		EnvironmentName: &m.EnvironmentName,
		LimitsMemory:    &m.LimitsMemory, LimitsMemorySwap: &m.LimitsMemorySwap,
		LimitsMemorySwappiness: &m.LimitsMemorySwappiness, LimitsMemoryReservation: &m.LimitsMemoryReservation,
		LimitsCPUs: &m.LimitsCPUs, LimitsCPUSet: &m.LimitsCPUSet, LimitsCPUShares: &m.LimitsCPUShares,
		HealthCheckEnabled: &m.HealthCheckEnabled, HealthCheckPath: &m.HealthCheckPath,
		HealthCheckPort: &m.HealthCheckPort, HealthCheckInterval: &m.HealthCheckInterval,
		HealthCheckTimeout: &m.HealthCheckTimeout, HealthCheckRetries: &m.HealthCheckRetries,
		HealthCheckStartPeriod: &m.HealthCheckStartPeriod,
		HealthCheckCommand:     &m.HealthCheckCommand, HealthCheckHost: &m.HealthCheckHost,
		HealthCheckMethod: &m.HealthCheckMethod, HealthCheckResponseText: &m.HealthCheckResponseText,
		HealthCheckReturnCode: &m.HealthCheckReturnCode, HealthCheckScheme: &m.HealthCheckScheme,
		HealthCheckType:     &m.HealthCheckType,
		IsAutoDeployEnabled: &m.IsAutoDeployEnabled,
		BaseDirectory:       &m.BaseDirectory, Dockerfile: &m.Dockerfile,
		DockerRegistryImageTag:          &m.DockerRegistryImageTag,
		DockerComposeDomains:            &m.DockerComposeDomains,
		DockerComposeLocation:           &m.DockerComposeLocation,
		DockerComposeCustomBuildCommand: &m.DockerComposeCustomBuildCommand,
		DockerComposeCustomStartCommand: &m.DockerComposeCustomStartCommand,
		GitCommitSha:                    &m.GitCommitSha, PublishDirectory: &m.PublishDirectory,
		WatchPaths: &m.WatchPaths, PreviewURLTemplate: &m.PreviewURLTemplate,
		CustomDockerRunOptions: &m.CustomDockerRunOptions, CustomLabels: &m.CustomLabels,
		CustomNetworkAliases: &m.CustomNetworkAliases, CustomNginxConfiguration: &m.CustomNginxConfiguration,
		PortsMappings: &m.PortsMappings, ConnectToDockerNetwork: &m.ConnectToDockerNetwork,
		Redirect: &m.Redirect, StaticImage: &m.StaticImage,
		IsStatic: &m.IsStatic, IsSPA: &m.IsSPA,
		IsForceHTTPSEnabled: &m.IsForceHTTPSEnabled, IsHTTPBasicAuthEnabled: &m.IsHTTPBasicAuthEnabled,
		HTTPBasicAuthUsername: &m.HTTPBasicAuthUsername, HTTPBasicAuthPassword: &m.HTTPBasicAuthPassword,
		PreDeploymentCommand: &m.PreDeploymentCommand, PreDeploymentCommandContainer: &m.PreDeploymentCommandContainer,
		PostDeploymentCommand: &m.PostDeploymentCommand, PostDeploymentCommandContainer: &m.PostDeploymentCommandContainer,
		ManualWebhookSecretBitbucket: &m.ManualWebhookSecretBitbucket, ManualWebhookSecretGitea: &m.ManualWebhookSecretGitea,
		ManualWebhookSecretGitHub: &m.ManualWebhookSecretGitHub, ManualWebhookSecretGitLab: &m.ManualWebhookSecretGitLab,
		ForceDomainOverride: &m.ForceDomainOverride, IsContainerLabelEscapeEnabled: &m.IsContainerLabelEscapeEnabled,
		IsPreserveRepositoryEnabled: &m.IsPreserveRepositoryEnabled, UseBuildServer: &m.UseBuildServer,
		// nil: GitRepository, GitBranch, BuildPack, PortsExposes, DockerfileLocation,
		//      InstallCommand, BuildCommand, StartCommand (not in compose model)
	}
}

// flattenDockerComposeApplication copies API fields into the Terraform state model.
func flattenDockerComposeApplication(app *client.Application, state *dockerComposeApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
	// Preserve user's docker_compose_raw from state: the API may return
	// a decoded or reformatted version that would cause a perpetual diff.
	if state.DockerComposeRaw.IsNull() || state.DockerComposeRaw.IsUnknown() {
		state.DockerComposeRaw = types.StringValue(app.DockerComposeRaw)
	}
}
