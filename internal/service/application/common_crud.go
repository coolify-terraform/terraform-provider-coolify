package application

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// setImportDefaults sets the default values for Computed+Default attributes
// during import. These must be set explicitly because Terraform does not apply
// schema defaults during import; the Read method relies on these initial values
// to avoid null-vs-default conflicts.
func setImportDefaults(ctx context.Context, resp *resource.ImportStateResponse) {
	set := func(attr string, v interface{}) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), v)...)
	}
	set("health_check_enabled", false)
	set("health_check_path", "/")
	set("health_check_interval", int64(5))
	set("health_check_timeout", int64(5))
	set("health_check_retries", int64(10))
	set("health_check_start_period", int64(5))
	set("is_auto_deploy_enabled", true)
	set("redirect", defaultRedirect)
	set("health_check_type", defaultHealthCheckType)
	set("health_check_method", defaultHealthCheckMeth)
	set("health_check_scheme", defaultHealthCheckSchm)
	set("health_check_return_code", defaultHealthCheckCode)
	set("health_check_host", defaultHealthCheckHost)
	set("static_image", defaultStaticImage)
	set("connect_to_docker_network", false)
	set("is_http_basic_auth_enabled", false)
	set("is_static", false)
	set("is_spa", false)
	set("is_force_https_enabled", true)
	set("is_container_label_escape_enabled", true)
	set("is_preserve_repository_enabled", false)
	set("use_build_server", false)
}

const applicationCreateReadBackFailedSummary = "Application created but refresh failed"

func addApplicationCreateReadBackDiagnostic(resp *resource.CreateResponse, detail string) {
	resp.Diagnostics.AddError(applicationCreateReadBackFailedSummary, detail)
}

func addApplicationCreateReadBackError(resp *resource.CreateResponse, uuid string, err error) {
	addApplicationCreateReadBackDiagnostic(
		resp,
		fmt.Sprintf(
			"Coolify created application %s, but the provider could not read it back: "+
				"Could not read application %s after create: %s. "+
				"The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.",
			uuid,
			uuid,
			err,
		),
	)
}

func addApplicationCreateReadBackNotFoundError(resp *resource.CreateResponse, uuid string) {
	addApplicationCreateReadBackDiagnostic(
		resp,
		fmt.Sprintf(
			"Coolify created application %s, but the provider could not read it back because the API returned 404 on the immediate read-back. "+
				"The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the application becomes readable through the API.",
			uuid,
		),
	)
}

// readBackAfterCreate reads the newly created application. If the immediate
// read-back fails, it leaves the partial state intact and records the failure.
func readBackAfterCreate(ctx context.Context, c *client.Client, uuid string, resp *resource.CreateResponse) *client.Application {
	app, err := c.GetApplication(ctx, uuid)
	if err == nil {
		return app
	}
	if client.IsNotFound(err) {
		addApplicationCreateReadBackNotFoundError(resp, uuid)
		return nil
	}
	addApplicationCreateReadBackError(resp, uuid, err)
	return nil
}

// updateAndReadBack performs the shared update-then-read pattern for all
// application resources. If redeployOnUpdate is true and runtime-affecting
// fields changed (common or type-specific), it automatically restarts the
// application after the update. Set typeSpecificFieldChanged to true when
// the caller detects changes to type-specific runtime fields (e.g.,
// docker_image for docker image apps, github_app_uuid for github app apps).
func updateAndReadBack(
	ctx context.Context,
	c *client.Client,
	uuid string,
	input client.UpdateApplicationInput,
	resp *resource.UpdateResponse,
	flatten func(*client.Application),
	redeployOnUpdate bool,
	plan, state commonAppFields,
	typeSpecificFieldChanged ...bool,
) {
	if _, err := c.UpdateApplication(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}

	extraChanged := len(typeSpecificFieldChanged) > 0 && typeSpecificFieldChanged[0]
	if redeployOnUpdate && (runtimeFieldsChanged(plan, state) || extraChanged) {
		tflog.Info(ctx, "runtime fields changed, restarting application", map[string]interface{}{"uuid": uuid})
		if _, err := c.RestartApplication(ctx, uuid); err != nil {
			resp.Diagnostics.AddWarning("Application updated but restart failed",
				fmt.Sprintf("The configuration was saved but the restart failed: %s. You may need to restart manually.", err))
		}
	}

	app, err := readApplicationAfterUpdate(ctx, c, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}
	flatten(app)
}

func readApplicationAfterUpdate(ctx context.Context, c *client.Client, uuid string) (*client.Application, error) {
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("reading application %s after update: %w", uuid, err)
	}
	return app, nil
}

// readApplication reads an application by UUID and calls the flatten function.
// If the application is not found, it removes the resource from state.
func readApplication(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.ReadResponse,
	flatten func(*client.Application),
) {
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	flatten(app)
}

// deleteApplication deletes an application by UUID and polls until the
// resource is fully removed. Coolify processes application deletions
// asynchronously via DeleteResourceJob; without polling, downstream
// resources (e.g. project) fail to delete because the app still exists.
// A 404 is treated as already-deleted and does not produce an error.
func deleteApplication(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.DeleteResponse,
) {
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	if err := c.DeleteApplication(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	if !client.PollUntilDeleted(ctx, func() error { _, err := c.GetApplication(ctx, uuid); return err }) {
		tflog.Warn(ctx, "resource may still exist after polling timeout", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
}

// importApplicationState validates the import ID and sets the initial state
// attributes common to all application resource types.
func importApplicationState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, compound, err := validate.ParseCompoundImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parsed.UUID)...)
	if compound {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parsed.ProjectUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parsed.ServerUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parsed.EnvironmentName)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	}
	setImportDefaults(ctx, resp)
	addApplicationImportSensitiveFieldsWarning(resp)
}

// addApplicationImportSensitiveFieldsWarning explains why imported application
// resources may show diffs for sensitive fields hidden by the API.
func addApplicationImportSensitiveFieldsWarning(resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides dockerfile, custom_labels, and docker_compose unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}
