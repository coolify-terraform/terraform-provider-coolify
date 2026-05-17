package storage

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &storageResource{}
	_ resource.ResourceWithConfigure   = &storageResource{}
	_ resource.ResourceWithImportState = &storageResource{}
)

// storageResource manages a persistent storage volume on a Coolify
// application, service, or database.
type storageResource struct {
	client *client.Client
}

// storageResourceModel maps the resource schema to Go types.
type storageResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	DatabaseUUID    types.String `tfsdk:"database_uuid"`
	ResourceUUID    types.String `tfsdk:"resource_uuid"`
	Name            types.String `tfsdk:"name"`
	MountPath       types.String `tfsdk:"mount_path"`
	HostPath        types.String `tfsdk:"host_path"`
}

// NewResource returns a new storageResource instance.
func NewResource() resource.Resource {
	return &storageResource{}
}

func (r *storageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (r *storageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a persistent storage volume on a Coolify application, service, or database.\n\n" +
			"~> **Note:** Each instance requires a List API call to read because the Coolify API does not " +
			"provide a singular GET endpoint for storage volumes. Large numbers of these resources " +
			"on a single parent resource may cause slower plan/apply times due to this API limitation.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the persistent storage.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to attach the storage to. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
						path.MatchRoot("database_uuid"),
					),
					validate.UUID(),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service that owns the storage. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. When set, `resource_uuid` must also be provided to identify which sub-resource within the service owns the storage. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database to attach the storage to. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the nested application or database inside a service. Required when `service_uuid` is set because Coolify services contain multiple sub-resources and the storage must target a specific one. Ignored for `application_uuid` and `database_uuid`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the persistent storage. Note: Coolify prepends an internal resource UUID to this name (e.g. `my-vol` becomes `{resource-uuid}-my-vol`).",
				Required:            true,
			},
			"mount_path": schema.StringAttribute{
				MarkdownDescription: "The mount path inside the container.",
				Required:            true,
			},
			"host_path": schema.StringAttribute{
				MarkdownDescription: "The host path to mount (optional; leave empty for a Docker volume).",
				Optional:            true,
			},
		},
	}
}

func (r *storageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// resolveParent determines the API parent type and UUID from the model.
func resolveParent(model *storageResourceModel) (parentType, parentUUID string, ok bool) {
	if !model.ApplicationUUID.IsNull() && !model.ApplicationUUID.IsUnknown() {
		return "applications", model.ApplicationUUID.ValueString(), true
	}
	if !model.ServiceUUID.IsNull() && !model.ServiceUUID.IsUnknown() {
		return "services", model.ServiceUUID.ValueString(), true
	}
	if !model.DatabaseUUID.IsNull() && !model.DatabaseUUID.IsUnknown() {
		return "databases", model.DatabaseUUID.ValueString(), true
	}
	return "", "", false
}

func (r *storageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_storage"})

	if !plan.ServiceUUID.IsNull() && !plan.ServiceUUID.IsUnknown() {
		if plan.ResourceUUID.IsNull() || plan.ResourceUUID.IsUnknown() {
			resp.Diagnostics.AddError(
				"Missing resource_uuid for service storage",
				"When creating storage on a service, resource_uuid must be set to the UUID of the nested application or database inside the service.",
			)
			return
		}
	}

	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	input := client.CreateStorageInput{
		Type:      "persistent",
		Name:      plan.Name.ValueString(),
		MountPath: plan.MountPath.ValueString(),
	}
	flex.SetIfKnown(&input.HostPath, plan.HostPath)
	flex.SetIfKnown(&input.ResourceUUID, plan.ResourceUUID)

	createResp, err := r.client.CreateStorage(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating persistent storage", fmt.Sprintf("storage on %s: %s", parentUUID, err))
		return
	}

	plan.UUID = types.StringValue(createResp.UUID)
	if plan.ResourceUUID.IsUnknown() {
		plan.ResourceUUID = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_storage", "uuid": state.UUID.ValueString()})

	parentType, parentUUID, ok := resolveParent(&state)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	storages, err := r.client.ListStorages(ctx, parentType, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading persistent storages", fmt.Sprintf("storage %s: %s", state.UUID.ValueString(), err))
		return
	}

	if !flattenStorageFromList(storages, &state) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_storage", "uuid": plan.UUID.ValueString()})

	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	input := client.UpdateStorageInput{
		UUID:      flex.StringValueOrNull(plan.UUID),
		Type:      "persistent",
		Name:      flex.StringIfChanged(plan.Name, state.Name),
		MountPath: flex.StringIfChanged(plan.MountPath, state.MountPath),
		HostPath:  flex.StringPtrForUpdate(plan.HostPath, state.HostPath),
	}

	err := r.client.UpdateStorage(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating persistent storage", fmt.Sprintf("storage %s: %s", plan.UUID.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_storage", "uuid": state.UUID.ValueString()})

	parentType, parentUUID, ok := resolveParent(&state)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	err := r.client.DeleteStorage(ctx, parentType, parentUUID, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting persistent storage", fmt.Sprintf("storage %s: %s", state.UUID.ValueString(), err))
		return
	}
}

// flattenStorageFromList finds the storage matching state.UUID in the list
// and updates the state model. Returns false if not found.
func flattenStorageFromList(storages []client.Storage, state *storageResourceModel) bool {
	for _, s := range storages {
		if s.UUID != state.UUID.ValueString() {
			continue
		}
		// Coolify prefixes storage names with an internal resource UUID
		// (for example, "resource-uuid-my-storage"). Preserve the
		// user's original name to avoid a perpetual diff.
		apiName := s.Name
		stateName := state.Name.ValueString()
		if stateName != "" && apiName != stateName && strings.HasSuffix(apiName, "-"+stateName) {
			apiName = stateName
		}
		state.Name = types.StringValue(apiName)
		state.MountPath = types.StringValue(s.MountPath)
		//nolint:gocritic // preserves null vs empty distinction for optional field
		if s.HostPath != "" {
			state.HostPath = types.StringValue(s.HostPath)
		} else if !state.HostPath.IsNull() {
			state.HostPath = types.StringNull()
		}
		if s.ResourceUUID != "" {
			state.ResourceUUID = types.StringValue(s.ResourceUUID)
		}
		return true
	}
	return false
}

func (r *storageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			`Expected "application:{app_uuid}:{storage_uuid}", "service:{svc_uuid}:{storage_uuid}", or "database:{db_uuid}:{storage_uuid}".`,
		)
		return
	}

	resourceType := parts[0]
	parentUUID := parts[1]
	storageUUID := parts[2]

	if err := validate.ImportUUID(parentUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "parent UUID segment: "+err.Error())
		return
	}
	if err := validate.ImportUUID(storageUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "storage UUID segment: "+err.Error())
		return
	}

	switch resourceType {
	case "application":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_uuid"), parentUUID)...)
	case "service":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_uuid"), parentUUID)...)
	case "database":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_uuid"), parentUUID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID type",
			fmt.Sprintf("Expected \"application\", \"service\", or \"database\", got %q.", resourceType),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), storageUUID)...)
}
