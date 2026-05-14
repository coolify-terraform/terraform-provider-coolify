package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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

type dockerfileApplicationResourceModel struct {
	applicationCommonModel
	GitRepository         types.String `tfsdk:"git_repository"`
	GitBranch             types.String `tfsdk:"git_branch"`
	BuildPack             types.String `tfsdk:"build_pack"`
	DockerfileLocation    types.String `tfsdk:"dockerfile_location"`
	BuildCommand          types.String `tfsdk:"build_command"`
	DockerfileTargetBuild types.String `tfsdk:"dockerfile_target_build"`
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
			"dockerfile_target_build": schema.StringAttribute{
				MarkdownDescription: "The target stage for multi-stage Docker builds.",
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

	app := readBackAfterCreate(ctx, r.client, created.UUID, resp)
	if app == nil {
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
	var state dockerfileApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_dockerfile_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common(), state.common())
	// Coolify v4 does not accept "dockerfile" on update; only "dockerfile_location" (a path).
	// For dockerfile apps, the content is set at creation time only, so clear the path field
	// to avoid sending base64 content as a path.
	input.DockerfileLocation = nil
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
	setImportDefaults(ctx, resp)
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides dockerfile, custom_labels, and docker_compose unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

func (m *dockerfileApplicationResourceModel) common() commonAppFields {
	c := m.applicationCommonModel.common()
	c.GitRepository = &m.GitRepository
	c.GitBranch = &m.GitBranch
	c.BuildPack = &m.BuildPack
	c.DockerfileLocation = &m.DockerfileLocation
	c.BuildCommand = &m.BuildCommand
	c.DockerfileTargetBuild = &m.DockerfileTargetBuild
	return c
}

func flattenDockerfileApplication(app *client.Application, state *dockerfileApplicationResourceModel) {
	// Save the user's dockerfile_location before the common flatten,
	// which may overwrite it with a stale value from the API's
	// dockerfile_location field. For dockerfile apps, the content
	// lives in the API's "dockerfile" field, not "dockerfile_location".
	savedDockerfileLocation := state.DockerfileLocation
	flattenApplicationCommon(app, state.common())
	// Preserve the user's value if it was set (normal CRUD flow).
	// On import, savedDockerfileLocation is null, so let the common
	// flatten's result stand (populated from app.DockerfileLocation).
	if !savedDockerfileLocation.IsNull() && !savedDockerfileLocation.IsUnknown() {
		state.DockerfileLocation = savedDockerfileLocation
	}
}
