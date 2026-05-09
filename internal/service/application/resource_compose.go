package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &DockerComposeApplicationResource{}
	_ resource.ResourceWithConfigure   = &DockerComposeApplicationResource{}
	_ resource.ResourceWithImportState = &DockerComposeApplicationResource{}
)

// DockerComposeApplicationResource manages a Coolify application deployed from a Docker Compose file.
type DockerComposeApplicationResource struct {
	client *client.Client
}

// DockerComposeApplicationResourceModel maps the resource schema to Go types.
type DockerComposeApplicationResourceModel struct {
	UUID             types.String   `tfsdk:"uuid"`
	Name             types.String   `tfsdk:"name"`
	Description      types.String   `tfsdk:"description"`
	ProjectUUID      types.String   `tfsdk:"project_uuid"`
	ServerUUID       types.String   `tfsdk:"server_uuid"`
	EnvironmentName  types.String   `tfsdk:"environment_name"`
	DockerComposeRaw types.String   `tfsdk:"docker_compose_raw"`
	FQDN             types.String   `tfsdk:"fqdn"`
	Status           types.String   `tfsdk:"status"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// NewDockerComposeResource returns a new DockerComposeApplicationResource instance.
func NewDockerComposeResource() resource.Resource {
	return &DockerComposeApplicationResource{}
}

func (r *DockerComposeApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_compose_application"
}

func (r *DockerComposeApplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify application deployed from a Docker Compose file.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the application.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the application.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the application.",
				Optional:            true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name for the application (defaults to `production`). Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("production"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "The raw Docker Compose YAML content.",
				Required:            true,
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must start with http:// or https://"),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
				Computed:            true,
			},
		},
	}
}

func (r *DockerComposeApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DockerComposeApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
	if v := flex.StringFromFramework(plan.EnvironmentName); v != "" {
		input.EnvironmentName = v
	}
	if v := flex.StringFromFramework(plan.Name); v != "" {
		input.Name = v
	}
	if v := flex.StringFromFramework(plan.Description); v != "" {
		input.Description = v
	}
	if v := flex.StringFromFramework(plan.FQDN); v != "" {
		input.FQDN = v
	}

	created, err := r.client.CreateDockerComposeApplication(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating docker compose application", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)

	app, err := r.client.GetApplication(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after creation", err.Error())
		return
	}

	flattenDockerComposeApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DockerComposeApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", err.Error())
		return
	}

	flattenDockerComposeApplication(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DockerComposeApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.UpdateApplicationInput{}
	strPtr := func(v types.String) *string {
		return flex.StringValueOrNull(v)
	}
	input.Name = strPtr(plan.Name)
	input.Description = strPtr(plan.Description)
	input.FQDN = strPtr(plan.FQDN)
	input.DockerComposeRaw = strPtr(plan.DockerComposeRaw)

	_, err := r.client.UpdateApplication(ctx, plan.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}

	app, err := r.client.GetApplication(ctx, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after update", err.Error())
		return
	}

	flattenDockerComposeApplication(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DockerComposeApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DockerComposeApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApplication(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", err.Error())
		return
	}
}

func (r *DockerComposeApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
}

// flattenDockerComposeApplication copies API fields into the Terraform state model.
func flattenDockerComposeApplication(app *client.Application, state *DockerComposeApplicationResourceModel) {
	state.UUID = types.StringValue(app.UUID)
	state.Name = types.StringValue(app.Name)
	state.Description = flex.StringToFramework(app.Description)
	state.DockerComposeRaw = flex.StringToFramework(app.DockerComposeRaw)
	state.FQDN = flex.StringToFramework(app.FQDN)

	state.ProjectUUID = flex.StringToFramework(app.ProjectUUID)
	state.ServerUUID = flex.StringToFramework(app.ServerUUID)
	state.Status = flex.StringToFramework(app.Status)
}
