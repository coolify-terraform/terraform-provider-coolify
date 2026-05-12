package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &gitHubAppApplicationResource{}
	_ resource.ResourceWithConfigure   = &gitHubAppApplicationResource{}
	_ resource.ResourceWithImportState = &gitHubAppApplicationResource{}
)

// gitHubAppApplicationResource manages a Coolify application deployed via a GitHub App.
type gitHubAppApplicationResource struct {
	client *client.Client
}

// gitHubAppApplicationResourceModel maps the resource schema to Go types.
type gitHubAppApplicationResourceModel struct {
	UUID                    types.String   `tfsdk:"uuid"`
	Name                    types.String   `tfsdk:"name"`
	Description             types.String   `tfsdk:"description"`
	ProjectUUID             types.String   `tfsdk:"project_uuid"`
	ServerUUID              types.String   `tfsdk:"server_uuid"`
	EnvironmentName         types.String   `tfsdk:"environment_name"`
	GitHubAppUUID           types.String   `tfsdk:"github_app_uuid"`
	GitRepository           types.String   `tfsdk:"git_repository"`
	GitBranch               types.String   `tfsdk:"git_branch"`
	BuildPack               types.String   `tfsdk:"build_pack"`
	PortsExposes            types.String   `tfsdk:"ports_exposes"`
	FQDN                    types.String   `tfsdk:"fqdn"`
	DockerfileLocation      types.String   `tfsdk:"dockerfile_location"`
	InstallCommand          types.String   `tfsdk:"install_command"`
	BuildCommand            types.String   `tfsdk:"build_command"`
	StartCommand            types.String   `tfsdk:"start_command"`
	Status                  types.String   `tfsdk:"status"`
	LimitsMemory            types.String   `tfsdk:"limits_memory"`
	LimitsMemorySwap        types.String   `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness  types.Int64    `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation types.String   `tfsdk:"limits_memory_reservation"`
	LimitsCPUs              types.String   `tfsdk:"limits_cpus"`
	LimitsCPUSet            types.String   `tfsdk:"limits_cpuset"`
	LimitsCPUShares         types.Int64    `tfsdk:"limits_cpu_shares"`
	HealthCheckEnabled      types.Bool     `tfsdk:"health_check_enabled"`
	HealthCheckPath         types.String   `tfsdk:"health_check_path"`
	HealthCheckPort         types.String   `tfsdk:"health_check_port"`
	HealthCheckInterval     types.Int64    `tfsdk:"health_check_interval"`
	HealthCheckTimeout      types.Int64    `tfsdk:"health_check_timeout"`
	HealthCheckRetries      types.Int64    `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod  types.Int64    `tfsdk:"health_check_start_period"`
	IsAutoDeployEnabled     types.Bool     `tfsdk:"is_auto_deploy_enabled"`
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
}

// NewGitHubAppResource returns a new gitHubAppApplicationResource instance.
func NewGitHubAppResource() resource.Resource {
	return &gitHubAppApplicationResource{}
}

func (r *gitHubAppApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_application"
}

func (r *gitHubAppApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed via a GitHub App integration.",
		Attributes: CommonAppAttrs(ctx, map[string]schema.Attribute{
			"github_app_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the GitHub App used for repository access.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git repository URL (e.g. `github.com/org/repo`).",
				Required:            true,
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch to deploy (defaults to `main`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("main"),
			},
			"build_pack": schema.StringAttribute{
				MarkdownDescription: "The build pack type. Valid values: `nixpacks`, `dockerfile`, `dockercompose`, `static`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("nixpacks", "dockerfile", "dockercompose", "static"),
				},
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `3000` or `3000,8080`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"3000\" or \"3000,8080\")"),
				},
			},
			"dockerfile_location": schema.StringAttribute{
				MarkdownDescription: "The path to the Dockerfile, relative to the repository root.",
				Optional:            true,
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the install phase.",
				Optional:            true,
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the build phase.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The command to run to start the application.",
				Optional:            true,
			},
		}),
	}
}

func (r *gitHubAppApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

//nolint:dupl // Create methods differ by input struct type and API call
func (r *gitHubAppApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_github_app_application"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateGitHubAppInput{
		ProjectUUID:   plan.ProjectUUID.ValueString(),
		ServerUUID:    plan.ServerUUID.ValueString(),
		GitHubAppUUID: plan.GitHubAppUUID.ValueString(),
		GitRepository: plan.GitRepository.ValueString(),
		BuildPack:     plan.BuildPack.ValueString(),
		PortsExposes:  plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.GitBranch, plan.GitBranch)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.DockerfileLocation, plan.DockerfileLocation)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateGitHubAppApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitHub App application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			resp.Diagnostics.AddError(
				"Application created but not persisted",
				fmt.Sprintf("Coolify returned UUID %s but the application was not found on read-back. "+
					"This usually means the target server is not reachable via SSH. "+
					"Verify the server is online and SSH-accessible before retrying.", created.UUID),
			)
			return
		}
		resp.Diagnostics.AddError("Error reading application after creation", fmt.Sprintf("application %s: %s", created.UUID, err))
		return
	}

	flattenGitHubAppApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_github_app_application", "uuid": state.UUID.ValueString()})

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenGitHubAppApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gitHubAppApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_github_app_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common())
	input.GitHubAppUUID = flex.StringValueOrNull(plan.GitHubAppUUID)
	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenGitHubAppApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitHubAppApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_github_app_application", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *gitHubAppApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("health_check_enabled"), false)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_auto_deploy_enabled"), true)...)
}

// flattenGitHubAppApplication copies API fields into the Terraform state model.
func (m *gitHubAppApplicationResourceModel) common() commonAppFields {
	return commonAppFields{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		GitRepository: &m.GitRepository, GitBranch: &m.GitBranch, BuildPack: &m.BuildPack,
		PortsExposes: &m.PortsExposes, FQDN: &m.FQDN, DockerfileLocation: &m.DockerfileLocation,
		InstallCommand: &m.InstallCommand, BuildCommand: &m.BuildCommand, StartCommand: &m.StartCommand,
		Status: &m.Status, ProjectUUID: &m.ProjectUUID, ServerUUID: &m.ServerUUID,
		EnvironmentName: &m.EnvironmentName,
		LimitsMemory:    &m.LimitsMemory, LimitsMemorySwap: &m.LimitsMemorySwap,
		LimitsMemorySwappiness: &m.LimitsMemorySwappiness, LimitsMemoryReservation: &m.LimitsMemoryReservation,
		LimitsCPUs: &m.LimitsCPUs, LimitsCPUSet: &m.LimitsCPUSet, LimitsCPUShares: &m.LimitsCPUShares,
		HealthCheckEnabled: &m.HealthCheckEnabled, HealthCheckPath: &m.HealthCheckPath,
		HealthCheckPort: &m.HealthCheckPort, HealthCheckInterval: &m.HealthCheckInterval,
		HealthCheckTimeout: &m.HealthCheckTimeout, HealthCheckRetries: &m.HealthCheckRetries,
		HealthCheckStartPeriod: &m.HealthCheckStartPeriod, IsAutoDeployEnabled: &m.IsAutoDeployEnabled,
	}
}

func flattenGitHubAppApplication(app *client.Application, state *gitHubAppApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
	if app.GitHubAppUUID != "" {
		state.GitHubAppUUID = types.StringValue(app.GitHubAppUUID)
	}
}
