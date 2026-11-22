package secretmanagersource

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = (*secretManagerSourceResource)(nil)
	_ resource.ResourceWithConfigure      = (*secretManagerSourceResource)(nil)
	_ resource.ResourceWithImportState    = (*secretManagerSourceResource)(nil)
	_ resource.ResourceWithValidateConfig = (*secretManagerSourceResource)(nil)
)

type secretManagerSourceResource struct {
	client *client.Client
}

func NewResource() resource.Resource { return &secretManagerSourceResource{} }

func (r *secretManagerSourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_manager_source"
}

func (r *secretManagerSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: resourceDescription,
		Attributes:          resourceAttributes(),
	}
}

func (r *secretManagerSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *secretManagerSourceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg model
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateParents(cfg.ApplicationUUID, cfg.ServiceUUID, cfg.DatabaseUUID, &resp.Diagnostics)
	validateSettings(cfg.settingFields, &resp.Diagnostics)
}

func (r *secretManagerSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	saved, ok := r.save(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &saved)...)
}

func (r *secretManagerSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !coolify46Ready(r.client, &resp.Diagnostics) {
		return
	}
	got, remove, ok := r.readSource(ctx, state.ApplicationUUID, state.ServiceUUID, state.DatabaseUUID, &resp.Diagnostics)
	if remove {
		resp.State.RemoveResource(ctx)
		return
	}
	if !ok {
		return
	}
	if !got.Linked {
		resp.State.RemoveResource(ctx)
		return
	}
	if !rejectMismatch(state.settingFields, got, &resp.Diagnostics) {
		return
	}
	if err := applySource(got, &state.IntegrationTokenUUID, &state.IntegrationProvider, &state.settingFields); err != nil {
		resp.Diagnostics.AddError("Error reading secret manager source", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *secretManagerSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	saved, ok := r.save(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &saved)...)
}

func (r *secretManagerSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !coolify46Ready(r.client, &resp.Diagnostics) {
		return
	}
	resourceType, uuid, ok := oneParent(state.ApplicationUUID, state.ServiceUUID, state.DatabaseUUID, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteSecretManagerSource(ctx, resourceType, uuid); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting secret manager source", fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
	}
}

func (r *secretManagerSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !coolify46Ready(r.client, &resp.Diagnostics) {
		return
	}
	attr, uuid, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), uuid)...)
}

func (r *secretManagerSourceResource) save(ctx context.Context, plan model, diags *diag.Diagnostics) (model, bool) {
	if !coolify46Ready(r.client, diags) {
		return plan, false
	}
	resourceType, uuid, ok := oneParent(plan.ApplicationUUID, plan.ServiceUUID, plan.DatabaseUUID, diags)
	if !ok {
		return plan, false
	}
	err := r.client.SetSecretManagerSource(ctx, resourceType, uuid, client.SetSecretManagerSourceInput{
		IntegrationTokenUUID: plan.IntegrationTokenUUID.ValueString(),
		Settings:             settingsBody(plan.settingFields),
	})
	if err != nil {
		diags.AddError("Error saving secret manager source", fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
		return plan, false
	}
	got, err := r.client.GetSecretManagerSource(ctx, resourceType, uuid)
	if err != nil {
		diags.AddError("Secret manager source saved but refresh failed", fmt.Sprintf("%s %s: %s. The next apply sends the same PATCH.", resourceType, uuid, err))
		return plan, false
	}
	if !got.Linked {
		diags.AddError("Secret manager source saved but refresh failed", fmt.Sprintf("%s %s is not linked.", resourceType, uuid))
		return plan, false
	}
	if !rejectMismatch(plan.settingFields, got, diags) {
		return plan, false
	}
	if err := applySource(got, &plan.IntegrationTokenUUID, &plan.IntegrationProvider, &plan.settingFields); err != nil {
		diags.AddError("Error reading secret manager source", err.Error())
		return plan, false
	}
	return plan, true
}

// readSource returns remove=true when the parent 404s. ok is false when an error was recorded.
func (r *secretManagerSourceResource) readSource(ctx context.Context, app, svc, db types.String, diags *diag.Diagnostics) (*client.SecretManagerSource, bool, bool) {
	resourceType, uuid, ok := oneParent(app, svc, db, diags)
	if !ok {
		return nil, false, false
	}
	got, err := r.client.GetSecretManagerSource(ctx, resourceType, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			return nil, true, false
		}
		diags.AddError("Error reading secret manager source", fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
		return nil, false, false
	}
	return got, false, true
}

func rejectMismatch(s settingFields, got *client.SecretManagerSource, diags *diag.Diagnostics) bool {
	attr, detail := storedMismatch(s, got.Settings)
	if attr == "" {
		return true
	}
	diags.AddAttributeError(path.Root(attr), "Coolify did not store this setting", detail)
	return false
}

func oneParent(app, svc, db types.String, diags *diag.Diagnostics) (string, string, bool) {
	if parentCount(app, svc, db) != 1 {
		diags.AddError("Exactly one parent", "Exactly one of application_uuid, service_uuid, or database_uuid must be set.")
		return "", "", false
	}
	resourceType, uuid, ok := parentOf(app, svc, db)
	if !ok {
		diags.AddError("Exactly one parent", "Exactly one of application_uuid, service_uuid, or database_uuid must be set.")
		return "", "", false
	}
	return resourceType, uuid, true
}

func parseImportID(id string) (string, string, error) {
	kind, uuid, ok := strings.Cut(id, ":")
	if !ok || strings.Contains(uuid, ":") {
		return "", "", fmt.Errorf("import ID %q must be application:<uuid>, service:<uuid>, or database:<uuid>", id)
	}
	attr := map[string]string{
		"application": "application_uuid",
		"service":     "service_uuid",
		"database":    "database_uuid",
	}[kind]
	if attr == "" {
		return "", "", fmt.Errorf("import ID %q must start with application, service, or database", id)
	}
	if err := validate.ImportUUID(uuid); err != nil {
		return "", "", err
	}
	return attr, uuid, nil
}
