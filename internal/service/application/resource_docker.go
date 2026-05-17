package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

type dockerImageApplicationResourceModel struct {
	applicationCommonModel
	DockerImage types.String `tfsdk:"docker_image"`
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
			"Unexpected Configure Type",
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
		resp.Diagnostics.AddError("Error creating docker image application",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	normalizeCommonAppCreateState(&plan.applicationCommonModel)

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
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_docker_image_application", "uuid": created.UUID})
}

func (r *dockerImageApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readApplication(ctx, r.client, "coolify_docker_image_application", state.UUID.ValueString(), resp, func(app *client.Application) {
		flattenDockerImageApplication(app, &state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	})
}

func (r *dockerImageApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state dockerImageApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_docker_image_application", "uuid": plan.UUID.ValueString()})

	input := buildUpdateInput(plan.common(), state.common())
	input.DockerRegistryImageName = flex.StringIfChanged(plan.DockerImage, state.DockerImage)

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
	deleteApplication(ctx, r.client, "coolify_docker_image_application", state.UUID.ValueString(), resp)
}

func (r *dockerImageApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importApplicationState(ctx, req, resp)
}

func (m *dockerImageApplicationResourceModel) common() commonAppFields {
	return m.applicationCommonModel.common()
}

// flattenDockerImageApplication copies API fields into the Terraform state model.
func flattenDockerImageApplication(app *client.Application, state *dockerImageApplicationResourceModel) {
	flattenApplicationCommon(app, state.common())
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
		// On import there is no prior state. Store the API value as-is;
		// if the tag was stripped, the user will see a diff on the next
		// plan and the update will re-set the image. We cannot reconstruct
		// the original tag from the stripped name alone.
		state.DockerImage = types.StringValue(app.DockerRegistryImageName)
	}
}
