package application

import (
	"context"
	"strings"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
}

// flattenApplicationCommon maps shared API fields into any application model
// via field pointers.
func flattenApplicationCommon(app *client.Application, f commonAppFields) {
	*f.UUID = types.StringValue(app.UUID)
	*f.Name = types.StringValue(app.Name)
	*f.Description = flex.StringToFramework(app.Description)
	// Coolify normalizes GitHub URLs by stripping the "https://github.com/"
	// prefix (e.g. "https://github.com/org/repo" becomes "org/repo"). Preserve
	// the user's original input if the API value is a suffix of it.
	if prior := f.GitRepository; !prior.IsNull() && !prior.IsUnknown() && strings.HasSuffix(prior.ValueString(), app.GitRepository) {
		*f.GitRepository = *prior
	} else {
		*f.GitRepository = types.StringValue(app.GitRepository)
	}
	*f.GitBranch = types.StringValue(app.GitBranch)
	*f.BuildPack = types.StringValue(app.BuildPack)
	// Coolify may override ports_exposes (e.g. return 80 instead of 3000
	// for Dockerfile apps). Preserve the user's configured value.
	if app.PortsExposes != "" {
		if f.PortsExposes.IsNull() || f.PortsExposes.IsUnknown() {
			*f.PortsExposes = types.StringValue(app.PortsExposes)
		}
	}
	*f.FQDN = flex.StringToFramework(app.FQDN)
	// Coolify does not return dockerfile_location on GET. Preserve from state.
	if app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	*f.InstallCommand = flex.StringToFramework(app.InstallCommand)
	*f.BuildCommand = flex.StringToFramework(app.BuildCommand)
	*f.StartCommand = flex.StringToFramework(app.StartCommand)
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
}

// buildUpdateInput constructs the shared UpdateApplicationInput from field pointers.
func buildUpdateInput(f commonAppFields) client.UpdateApplicationInput {
	strPtr := flex.StringValueOrNull
	return client.UpdateApplicationInput{
		Name:               strPtr(*f.Name),
		Description:        strPtr(*f.Description),
		GitRepository:      strPtr(*f.GitRepository),
		GitBranch:          strPtr(*f.GitBranch),
		BuildPack:          strPtr(*f.BuildPack),
		PortsExposes:       strPtr(*f.PortsExposes),
		FQDN:               strPtr(*f.FQDN),
		DockerfileLocation: strPtr(*f.DockerfileLocation),
		InstallCommand:     strPtr(*f.InstallCommand),
		BuildCommand:       strPtr(*f.BuildCommand),
		StartCommand:       strPtr(*f.StartCommand),
	}
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
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application after update", err.Error())
		return
	}
	flatten(app)
}
