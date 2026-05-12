package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dockerImageApplicationResource{}
	_ resource.ResourceWithConfigure   = &dockerImageApplicationResource{}
	_ resource.ResourceWithImportState = &dockerImageApplicationResource{}
)

// dockerImageApplicationResource manages a Coolify application deployed from a Docker image.
type dockerImageApplicationResource struct {
	client *client.Client
}

// dockerImageApplicationResourceModel maps the resource schema to Go types.
type dockerImageApplicationResourceModel struct {
	UUID                    types.String   `tfsdk:"uuid"`
	Name                    types.String   `tfsdk:"name"`
	Description             types.String   `tfsdk:"description"`
	ProjectUUID             types.String   `tfsdk:"project_uuid"`
	ServerUUID              types.String   `tfsdk:"server_uuid"`
	EnvironmentName         types.String   `tfsdk:"environment_name"`
	DockerImage             types.String   `tfsdk:"docker_image"`
	PortsExposes            types.String   `tfsdk:"ports_exposes"`
	FQDN                    types.String   `tfsdk:"fqdn"`
	InstallCommand          types.String   `tfsdk:"install_command"`
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

// NewDockerResource returns a new dockerImageApplicationResource instance.
func NewDockerResource() resource.Resource {
	return &dockerImageApplicationResource{}
}

func (r *dockerImageApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_image_application"
}

func (r *dockerImageApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Docker image.",
		Attributes: CommonAppAttrs(ctx, map[string]schema.Attribute{
			"docker_image": schema.StringAttribute{
				MarkdownDescription: "The Docker image to deploy (e.g. `nginx:latest`, `ghcr.io/org/app:v1`). Note: Coolify strips image tags internally (e.g. `redis:7-alpine` is stored as `redis`). The provider preserves your configured value.",
				Required:            true,
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `80` or `80,443`). Note: Coolify may override this value internally; the provider preserves your configured value.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"80\" or \"80,443\")"),
				},
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The command to run during the install phase.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The command to run to start the application.",
				Optional:            true,
			},
		}),
	}
}

func (r *dockerImageApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dockerImageApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_docker_image_application"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input := client.CreateDockerImageAppInput{
		ProjectUUID:  plan.ProjectUUID.ValueString(),
		ServerUUID:   plan.ServerUUID.ValueString(),
		DockerImage:  plan.DockerImage.ValueString(),
		PortsExposes: plan.PortsExposes.ValueString(),
	}
	flex.SetIfKnown(&input.EnvironmentName, plan.EnvironmentName)
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.FQDN, plan.FQDN)
	flex.SetIfKnown(&input.InstallCommand, plan.InstallCommand)
	flex.SetIfKnown(&input.StartCommand, plan.StartCommand)

	created, err := r.client.CreateDockerImageApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating docker image application", err.Error())
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

	flattenDockerImageApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerImageApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_docker_image_application", "uuid": state.UUID.ValueString()})

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenDockerImageApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dockerImageApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_docker_image_application", "uuid": plan.UUID.ValueString()})

	input := client.UpdateApplicationInput{}
	strPtr := flex.StringValueOrNull
	input.Name = strPtr(plan.Name)
	input.Description = strPtr(plan.Description)
	input.PortsExposes = strPtr(plan.PortsExposes)
	input.FQDN = strPtr(plan.FQDN)
	input.InstallCommand = strPtr(plan.InstallCommand)
	input.StartCommand = strPtr(plan.StartCommand)
	input.DockerRegistryImageName = strPtr(plan.DockerImage)
	input.LimitsMemory = strPtr(plan.LimitsMemory)
	input.LimitsMemorySwap = strPtr(plan.LimitsMemorySwap)
	input.LimitsMemorySwappiness = flex.Int64PtrFromFramework(plan.LimitsMemorySwappiness)
	input.LimitsMemoryReservation = strPtr(plan.LimitsMemoryReservation)
	input.LimitsCPUs = strPtr(plan.LimitsCPUs)
	input.LimitsCPUSet = strPtr(plan.LimitsCPUSet)
	input.LimitsCPUShares = flex.Int64PtrFromFramework(plan.LimitsCPUShares)
	input.HealthCheckEnabled = flex.BoolValueOrNull(plan.HealthCheckEnabled)
	input.HealthCheckPath = strPtr(plan.HealthCheckPath)
	input.HealthCheckPort = strPtr(plan.HealthCheckPort)
	input.HealthCheckInterval = flex.Int64PtrFromFramework(plan.HealthCheckInterval)
	input.HealthCheckTimeout = flex.Int64PtrFromFramework(plan.HealthCheckTimeout)
	input.HealthCheckRetries = flex.Int64PtrFromFramework(plan.HealthCheckRetries)
	input.HealthCheckStartPeriod = flex.Int64PtrFromFramework(plan.HealthCheckStartPeriod)
	input.IsAutoDeployEnabled = flex.BoolValueOrNull(plan.IsAutoDeployEnabled)

	updateAndReadBack(ctx, r.client, plan.UUID.ValueString(), input, resp, func(app *client.Application) {
		flattenDockerImageApplication(app, &plan)
	})
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dockerImageApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_docker_image_application", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *dockerImageApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("health_check_enabled"), false)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_auto_deploy_enabled"), true)...)
}

// flattenDockerImageApplication copies API fields into the Terraform state model.
func flattenDockerImageApplication(app *client.Application, state *dockerImageApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	// Coolify may strip the tag from Docker image names (e.g.
	// "redis:7-alpine" becomes "redis"). Preserve the user's original value
	// if the API value matches the image name without the tag.
	if prior := state.DockerImage; !prior.IsNull() && !prior.IsUnknown() {
		priorVal := prior.ValueString()
		apiVal := app.DockerRegistryImageName
		if priorVal == apiVal || strings.SplitN(priorVal, ":", 2)[0] == apiVal {
			// keep existing state value (user's image:tag is preserved)
		} else {
			state.DockerImage = types.StringValue(apiVal)
		}
	} else {
		state.DockerImage = types.StringValue(app.DockerRegistryImageName)
	}
	// Coolify may override ports_exposes. Preserve the user's configured value.
	if state.PortsExposes.IsNull() || state.PortsExposes.IsUnknown() {
		state.PortsExposes = types.StringValue(app.PortsExposes)
	}
	state.FQDN = flex.StringToFramework(app.FQDN)
	state.InstallCommand = flex.StringToFramework(app.InstallCommand)
	state.StartCommand = flex.StringToFramework(app.StartCommand)
	state.Status = flex.StringToFramework(app.Status)

	if app.ProjectUUID != "" {
		state.ProjectUUID = types.StringValue(app.ProjectUUID)
	}
	if app.ServerUUID != "" {
		state.ServerUUID = types.StringValue(app.ServerUUID)
	}
	if app.EnvironmentName != "" {
		state.EnvironmentName = flex.StringToFramework(app.EnvironmentName)
	}
	flattenLimitsAndHealth(app, commonAppFields{
		LimitsMemory: &state.LimitsMemory, LimitsMemorySwap: &state.LimitsMemorySwap,
		LimitsMemorySwappiness: &state.LimitsMemorySwappiness, LimitsMemoryReservation: &state.LimitsMemoryReservation,
		LimitsCPUs: &state.LimitsCPUs, LimitsCPUSet: &state.LimitsCPUSet, LimitsCPUShares: &state.LimitsCPUShares,
		HealthCheckEnabled: &state.HealthCheckEnabled, HealthCheckPath: &state.HealthCheckPath,
		HealthCheckPort: &state.HealthCheckPort, HealthCheckInterval: &state.HealthCheckInterval,
		HealthCheckTimeout: &state.HealthCheckTimeout, HealthCheckRetries: &state.HealthCheckRetries,
		HealthCheckStartPeriod: &state.HealthCheckStartPeriod, IsAutoDeployEnabled: &state.IsAutoDeployEnabled,
	})
	// Auto-deploy
	if app.IsAutoDeployEnabled != nil {
		state.IsAutoDeployEnabled = types.BoolValue(*app.IsAutoDeployEnabled)
	}
}
