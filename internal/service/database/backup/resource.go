package backup

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &databaseBackupResource{}
	_ resource.ResourceWithConfigure   = &databaseBackupResource{}
	_ resource.ResourceWithImportState = &databaseBackupResource{}
)

type databaseBackupResource struct {
	client *client.Client
}

type databaseBackupResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	UUID         types.String `tfsdk:"uuid"`
	DatabaseUUID types.String `tfsdk:"database_uuid"`
	Frequency    types.String `tfsdk:"frequency"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	S3StorageID  types.String `tfsdk:"s3_storage_id"`
	RetainDays   types.Int64  `tfsdk:"retain_days"`
}

func NewResource() resource.Resource { return &databaseBackupResource{} }

func (r *databaseBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_backup"
}

func (r *databaseBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a scheduled database backup configuration on Coolify.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric ID of the backup configuration.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the backup configuration.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database to back up. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Cron expression for backup schedule (e.g. `0 2 * * *` for daily at 2 AM, or `@daily`, `@hourly`, `@weekly`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g. \"0 2 * * *\" or \"@daily\")",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the backup schedule is active.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"s3_storage_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the S3 storage destination for off-site backups. Use the `uuid` output of a `coolify_s3_storage` resource. When omitted, backups are stored locally on the server.",
				Optional:            true,
			},
			"retain_days": schema.Int64Attribute{
				MarkdownDescription: "Number of backup copies to retain locally (not days). For example, `7` keeps the last 7 backups regardless of their age.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
	}
}

func (r *databaseBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *databaseBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateDatabaseBackupInput{
		Frequency: plan.Frequency.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	}
	flex.SetIfKnown(&input.S3StorageID, plan.S3StorageID)
	flex.SetInt64Ptr(&input.RetainDays, plan.RetainDays)

	created, err := r.client.CreateDatabaseBackup(ctx, plan.DatabaseUUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database backup", err.Error())
		return
	}

	// Save the user's planned enabled value before flattening, because
	// Coolify may return enabled=false immediately after creation even
	// when true was sent. The next Read will pick up the real value.
	plannedEnabled := plan.Enabled
	flattenDatabaseBackup(created, &plan)
	if plannedEnabled.ValueBool() && !plan.Enabled.ValueBool() {
		plan.Enabled = plannedEnabled
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	b, err := r.client.GetDatabaseBackup(ctx, state.DatabaseUUID.ValueString(), int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading database backup", err.Error())
		return
	}

	flattenDatabaseBackup(b, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbUUID := state.DatabaseUUID.ValueString()
	backupID := int(state.ID.ValueInt64())

	input := client.UpdateDatabaseBackupInput{}
	flex.SetStrPtr(&input.Frequency, plan.Frequency)
	flex.SetBoolPtr(&input.Enabled, plan.Enabled)
	input.S3StorageID = flex.StringPtrForUpdate(plan.S3StorageID, state.S3StorageID)
	input.RetainDays = flex.Int64PtrForUpdate(plan.RetainDays, state.RetainDays)

	if _, err := r.client.UpdateDatabaseBackup(ctx, dbUUID, backupID, input); err != nil {
		resp.Diagnostics.AddError("Error updating database backup", err.Error())
		return
	}

	b, err := r.client.GetDatabaseBackup(ctx, dbUUID, backupID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading database backup after update", err.Error())
		return
	}

	flattenDatabaseBackup(b, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDatabaseBackup(ctx, state.DatabaseUUID.ValueString(), int(state.ID.ValueInt64())); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting database backup", err.Error())
		return
	}
}

func (r *databaseBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			"Expected \"database_uuid:backup_id\".",
		)
		return
	}

	dbUUID := parts[0]
	if err := validate.ImportUUID(dbUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "database UUID segment: "+err.Error())
		return
	}
	backupID, err := strconv.Atoi(parts[1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("backup_id must be an integer, got %q.", parts[1]),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_uuid"), dbUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), int64(backupID))...)
}

func flattenDatabaseBackup(b *client.DatabaseBackup, m *databaseBackupResourceModel) {
	m.ID = types.Int64Value(int64(b.ID))
	m.UUID = types.StringValue(b.UUID)
	if b.DatabaseUUID != "" {
		m.DatabaseUUID = types.StringValue(b.DatabaseUUID)
	}
	if b.Frequency != "" {
		m.Frequency = types.StringValue(b.Frequency)
	}
	m.Enabled = types.BoolValue(b.Enabled)
	m.S3StorageID = flex.StringToFramework(b.S3StorageID)
	switch {
	case b.RetainDays != nil:
		m.RetainDays = types.Int64Value(*b.RetainDays)
	case !m.RetainDays.IsNull():
		// Preserve existing state value when API returns nil
	default:
		m.RetainDays = types.Int64Null()
	}
}
