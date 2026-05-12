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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dockerfileApplicationResource{}
	_ resource.ResourceWithConfigure   = &dockerfileApplicationResource{}
	_ resource.ResourceWithImportState = &dockerfileApplicationResource{}
)

// dockerfileApplicationResource manages a Coolify application deployed from a Dockerfile.
type dockerfileApplicationResource struct {
	client *client.Client
}

// dockerfileApplicationResourceModel maps the resource schema to Go types.
type dockerfileApplicationResourceModel struct {
	UUID                    types.String   `tfsdk:"uuid"`
	Name                    types.String   `tfsdk:"name"`
	Description             types.String   `tfsdk:"description"`
	ProjectUUID             types.String   `tfsdk:"project_uuid"`
	ServerUUID              types.String   `tfsdk:"server_uuid"`
	EnvironmentName         types.String   `tfsdk:"environment_name"`
	DockerfileLocation      types.String   `tfsdk:"dockerfile_location"`
	PortsExposes            types.String   `tfsdk:"ports_exposes"`
	FQDN                    types.String   `tfsdk:"fqdn"`
	InstallCommand          types.String   `tfsdk:"install_command"`
	BuildCommand            types.String   `tfsdk:"build_command"`
	StartCommand            types.String   `tfsdk:"start_command"`
	GitRepository           types.String   `tfsdk:"git_repository"`
	GitBranch               types.String   `tfsdk:"git_branch"`
	BuildPack               types.String   `tfsdk:"build_pack"`
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

// NewDockerfileResource returns a new dockerfileApplicationResource instance.
func NewDockerfileResource() resource.Resource {
	return &dockerfileApplicationResource{}
}

func (r *dockerfileApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dockerfile_application"
}

func (r *dockerfileApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Dockerfile.",
		Attributes: CommonAppAttrs(ctx, map[string]schema.Attribute{
			"dockerfile_location": schema.StringAttribute{
				MarkdownDescription: "The Dockerfile content, **base64-encoded**. Use `base64encode(<<-DOCKERFILE ... DOCKERFILE)` in your configuration. Despite the field name, this is not a file path.",
				Required:            true,
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `80` or `80,443`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"80\" or \"80,443\")"),
				},
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
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git repository URL. Read-only, set by the API.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch. Read-only, set by the API.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"build_pack": schema.StringAttribute{
				MarkdownDescription: "The build pack type. Read-only, set by the API.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		}),
	}
}

func (r *dockerfileApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dockerfileApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_dockerfile_application"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateDockerfileAppInput{
		ProjectUUID:        plan.ProjectUUID.ValueString(),
		ServerUUID:         plan.ServerUUID.ValueString(),
		DockerfileLocation: plan.DockerfileLocation.ValueString(),
		PortsExposes:       plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.BuildCommand, plan.BuildCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateDockerfileApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dockerfile application", err.Error())
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

	flattenDockerfileApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerfileApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_dockerfile_application", "uuid": state.UUID.ValueString()})

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenDockerfileApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerfileApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_dockerfile_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common())
	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenDockerfileApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerfileApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_dockerfile_application", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *dockerfileApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("health_check_enabled"), false)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_auto_deploy_enabled"), true)...)
}

// flattenDockerfileApplication copies API fields into the Terraform state model.
func (m *dockerfileApplicationResourceModel) common() commonAppFields {
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

func flattenDockerfileApplication(app *client.Application, state *dockerfileApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
}
