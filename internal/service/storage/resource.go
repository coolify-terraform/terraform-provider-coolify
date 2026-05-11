package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		MarkdownDescription: "Manages a persistent storage volume on a Coolify application, service, or database.",
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
				MarkdownDescription: "The UUID of the service to attach the storage to. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the persistent storage.",
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
			"Unexpected Resource Configure Type",
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

	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	input := client.CreateStorageInput{
		Name:      plan.Name.ValueString(),
		MountPath: plan.MountPath.ValueString(),
	}
	flex.SetIfKnown(&input.HostPath, plan.HostPath)

	createResp, err := r.client.CreateStorage(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating persistent storage", err.Error())
		return
	}

	plan.UUID = types.StringValue(createResp.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
		resp.Diagnostics.AddError("Error reading persistent storages", err.Error())
		return
	}

	found := false
	for _, s := range storages {
		if s.UUID == state.UUID.ValueString() {
			state.Name = types.StringValue(s.Name)
			state.MountPath = types.StringValue(s.MountPath)
			//nolint:gocritic // preserves null vs empty distinction for optional field
			if s.HostPath != "" {
				state.HostPath = types.StringValue(s.HostPath)
			} else if state.HostPath.IsNull() {
				// keep null if it was null before and API returns empty
			} else {
				state.HostPath = types.StringNull()
			}
			found = true
			break
		}
	}

	if !found {
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

	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	input := client.UpdateStorageInput{
		UUID:      flex.StringValueOrNull(plan.UUID),
		Name:      flex.StringValueOrNull(plan.Name),
		MountPath: flex.StringValueOrNull(plan.MountPath),
		HostPath:  flex.StringPtrForUpdate(plan.HostPath, state.HostPath),
	}

	err := r.client.UpdateStorage(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating persistent storage", err.Error())
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
		resp.Diagnostics.AddError("Error deleting persistent storage", err.Error())
		return
	}
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
