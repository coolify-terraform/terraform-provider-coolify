package backup

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &databaseBackupResource{}
	_ resource.ResourceWithConfigure      = &databaseBackupResource{}
	_ resource.ResourceWithImportState    = &databaseBackupResource{}
	_ resource.ResourceWithUpgradeState   = &databaseBackupResource{}
	_ resource.ResourceWithValidateConfig = &databaseBackupResource{}
)

type databaseBackupResource struct {
	client *client.Client
}

type databaseBackupResourceModel struct {
	ID                    types.Int64  `tfsdk:"id"`
	UUID                  types.String `tfsdk:"uuid"`
	DatabaseUUID          types.String `tfsdk:"database_uuid"`
	Frequency             types.String `tfsdk:"frequency"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	SaveS3                types.Bool   `tfsdk:"save_s3"`
	S3StorageUUID         types.String `tfsdk:"s3_storage_uuid"`
	DatabasesToBackup     types.String `tfsdk:"databases_to_backup"`
	DumpAll               types.Bool   `tfsdk:"dump_all"`
	BackupNow             types.Bool   `tfsdk:"backup_now"`
	RetainAmountLocally   types.Int64  `tfsdk:"retain_amount_locally"`
	RetainDaysLocally     types.Int64  `tfsdk:"retain_days_locally"`
	RetainMaxStorageLocal types.Int64  `tfsdk:"retain_max_storage_locally"`
	RetainAmountS3        types.Int64  `tfsdk:"retain_amount_s3"`
	RetainDaysS3          types.Int64  `tfsdk:"retain_days_s3"`
	RetainMaxStorageS3    types.Int64  `tfsdk:"retain_max_storage_s3"`
	Timeout               types.Int64  `tfsdk:"timeout"`
}

func NewResource() resource.Resource { return &databaseBackupResource{} }

func (r *databaseBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_backup"
}

func (r *databaseBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
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
				MarkdownDescription: "Cron expression for backup schedule (e.g., `0 2 * * *` for daily at 2 AM, or `@daily`, `@hourly`, `@weekly`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g., \"0 2 * * *\" or \"@daily\")",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the backup schedule is active.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"save_s3": schema.BoolAttribute{
				MarkdownDescription: "Whether to save backups to S3 storage.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the S3 storage destination for off-site backups. Required when `save_s3` is `true`.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"databases_to_backup": schema.StringAttribute{
				MarkdownDescription: "Comma-separated list of database names to back up selectively. Defaults to the primary database name if not specified.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dump_all": schema.BoolAttribute{
				MarkdownDescription: "Whether to dump all databases.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"backup_now": schema.BoolAttribute{
				MarkdownDescription: "Trigger an immediate backup after creation. Only used during create, ignored on updates.",
				Optional:            true,
			},
			"retain_amount_locally": schema.Int64Attribute{
				MarkdownDescription: "Number of backup copies to retain locally.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retain_days_locally": schema.Int64Attribute{
				MarkdownDescription: "Number of days to retain backups locally.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retain_max_storage_locally": schema.Int64Attribute{
				MarkdownDescription: "Maximum storage in MB for local backups.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retain_amount_s3": schema.Int64Attribute{
				MarkdownDescription: "Number of backup copies to retain in S3.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retain_days_s3": schema.Int64Attribute{
				MarkdownDescription: "Number of days to retain backups in S3.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retain_max_storage_s3": schema.Int64Attribute{
				MarkdownDescription: "Maximum storage in MB for S3 backups.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Backup job timeout in seconds (60-36000).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.Between(60, 36000)},
			},
		},
	}
}

func (r *databaseBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *databaseBackupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config databaseBackupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.SaveS3.IsNull() && !config.SaveS3.IsUnknown() && config.SaveS3.ValueBool() {
		if config.S3StorageUUID.IsNull() || config.S3StorageUUID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("s3_storage_uuid"),
				"Missing S3 Storage UUID",
				"`s3_storage_uuid` must be set when `save_s3` is `true`.",
			)
		}
	}
}

func (r *databaseBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_backup"})

	input := client.CreateDatabaseBackupInput{
		Frequency: plan.Frequency.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	}
	flex.SetBoolPtr(&input.SaveS3, plan.SaveS3)
	flex.SetIfKnown(&input.S3StorageID, plan.S3StorageUUID)
	flex.SetIfKnown(&input.DatabasesToBackup, plan.DatabasesToBackup)
	flex.SetBoolPtr(&input.DumpAll, plan.DumpAll)
	flex.SetBoolPtr(&input.BackupNow, plan.BackupNow)
	input.RetainAmountLocally = flex.Int64PtrFromFramework(plan.RetainAmountLocally)
	input.RetainDaysLocally = flex.Int64PtrFromFramework(plan.RetainDaysLocally)
	input.RetainMaxStorageLocal = flex.Float64PtrFromInt64Framework(plan.RetainMaxStorageLocal)
	input.RetainAmountS3 = flex.Int64PtrFromFramework(plan.RetainAmountS3)
	input.RetainDaysS3 = flex.Int64PtrFromFramework(plan.RetainDaysS3)
	input.RetainMaxStorageS3 = flex.Float64PtrFromInt64Framework(plan.RetainMaxStorageS3)
	input.Timeout = flex.Int64PtrFromFramework(plan.Timeout)

	created, err := r.client.CreateDatabaseBackup(ctx, plan.DatabaseUUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database backup", fmt.Sprintf("backup for database %s: %s", plan.DatabaseUUID.ValueString(), err))
		return
	}

	// The Coolify Create endpoint returns only {"uuid", "message"}, not
	// the full backup object. Save partial state immediately so the
	// resource is tracked even if the follow-up read fails.
	plan.UUID = types.StringValue(created.UUID)
	if plan.ID.IsUnknown() {
		plan.ID = types.Int64Null()
	}
	if plan.Enabled.IsUnknown() {
		plan.Enabled = types.BoolNull()
	}
	if plan.SaveS3.IsUnknown() {
		plan.SaveS3 = types.BoolNull()
	}
	if plan.DumpAll.IsUnknown() {
		plan.DumpAll = types.BoolNull()
	}
	if plan.RetainAmountLocally.IsUnknown() {
		plan.RetainAmountLocally = types.Int64Null()
	}
	if plan.RetainDaysLocally.IsUnknown() {
		plan.RetainDaysLocally = types.Int64Null()
	}
	if plan.RetainMaxStorageLocal.IsUnknown() {
		plan.RetainMaxStorageLocal = types.Int64Null()
	}
	if plan.RetainAmountS3.IsUnknown() {
		plan.RetainAmountS3 = types.Int64Null()
	}
	if plan.RetainDaysS3.IsUnknown() {
		plan.RetainDaysS3 = types.Int64Null()
	}
	if plan.RetainMaxStorageS3.IsUnknown() {
		plan.RetainMaxStorageS3 = types.Int64Null()
	}
	if plan.Timeout.IsUnknown() {
		plan.Timeout = types.Int64Null()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dbUUID := plan.DatabaseUUID.ValueString()

	// Resolve the real numeric ID by listing backups and matching by UUID,
	// then do a full GET to populate all fields.
	backups, listErr := r.client.ListDatabaseBackups(ctx, dbUUID)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Database backup created but refresh failed",
			fmt.Sprintf("Coolify created database backup %s for database %s, but the provider could not read it back: Could not list database backups for %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, dbUUID, dbUUID, listErr),
		)
		return
	}
	var found *client.DatabaseBackup
	for i := range backups {
		if backups[i].UUID == created.UUID {
			found = &backups[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError(
			"Database backup created but refresh failed",
			fmt.Sprintf("Coolify created database backup %s for database %s, but the provider could not read it back: Could not resolve backup %s from the database %s backup list after create. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", created.UUID, dbUUID, created.UUID, dbUUID),
		)
		return
	}

	flattenDatabaseBackup(found, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": created.UUID})
}

func (r *databaseBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})

	dbUUID := state.DatabaseUUID.ValueString()
	backupID := int(state.ID.ValueInt64())

	b, readErr := r.readBackup(ctx, dbUUID, backupID, state.UUID)
	if readErr != nil {
		if client.IsNotFound(readErr) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading database backup", fmt.Sprintf("backup %d for database %s: %s", backupID, dbUUID, readErr))
		return
	}
	if b == nil {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})
		resp.State.RemoveResource(ctx)
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})

	dbUUID := state.DatabaseUUID.ValueString()
	backupID := int(state.ID.ValueInt64())

	input := client.UpdateDatabaseBackupInput{
		Frequency:             flex.StringIfChanged(plan.Frequency, state.Frequency),
		Enabled:               flex.BoolIfChanged(plan.Enabled, state.Enabled),
		SaveS3:                flex.BoolIfChanged(plan.SaveS3, state.SaveS3),
		S3StorageID:           flex.StringPtrForUpdate(plan.S3StorageUUID, state.S3StorageUUID),
		DatabasesToBackup:     flex.StringPtrForUpdate(plan.DatabasesToBackup, state.DatabasesToBackup),
		DumpAll:               flex.BoolIfChanged(plan.DumpAll, state.DumpAll),
		RetainAmountLocally:   flex.Int64IfChanged(plan.RetainAmountLocally, state.RetainAmountLocally),
		RetainDaysLocally:     flex.Int64IfChanged(plan.RetainDaysLocally, state.RetainDaysLocally),
		RetainMaxStorageLocal: flex.Float64IfChangedFromInt64(plan.RetainMaxStorageLocal, state.RetainMaxStorageLocal),
		RetainAmountS3:        flex.Int64IfChanged(plan.RetainAmountS3, state.RetainAmountS3),
		RetainDaysS3:          flex.Int64IfChanged(plan.RetainDaysS3, state.RetainDaysS3),
		RetainMaxStorageS3:    flex.Float64IfChangedFromInt64(plan.RetainMaxStorageS3, state.RetainMaxStorageS3),
		Timeout:               flex.Int64IfChanged(plan.Timeout, state.Timeout),
	}

	if _, err := r.client.UpdateDatabaseBackup(ctx, dbUUID, state.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating database backup", fmt.Sprintf("backup %s for database %s: %s", state.UUID.ValueString(), dbUUID, err))
		return
	}

	b, err := r.readBackup(ctx, dbUUID, backupID, state.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading database backup after update", fmt.Sprintf("backup %d for database %s: %s", backupID, dbUUID, err))
		return
	}
	if b == nil {
		resp.Diagnostics.AddError("Error reading database backup after update", fmt.Sprintf("backup %d not found for database %s", backupID, dbUUID))
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
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})

	if err := r.client.DeleteDatabaseBackup(ctx, state.DatabaseUUID.ValueString(), state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting database backup", fmt.Sprintf("backup %d for database %s: %s", int(state.ID.ValueInt64()), state.DatabaseUUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_database_backup", "uuid": state.UUID.ValueString()})
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
	if backupID <= 0 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("backup_id must be a positive integer, got %d.", backupID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_uuid"), dbUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), int64(backupID))...)
}

func (r *databaseBackupResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			// Version 0 -> 1: rename s3_storage_id to s3_storage_uuid.
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                         schema.Int64Attribute{Computed: true},
					"uuid":                       schema.StringAttribute{Computed: true},
					"database_uuid":              schema.StringAttribute{Required: true},
					"frequency":                  schema.StringAttribute{Optional: true},
					"enabled":                    schema.BoolAttribute{Optional: true, Computed: true},
					"save_s3":                    schema.BoolAttribute{Optional: true, Computed: true},
					"s3_storage_id":              schema.StringAttribute{Optional: true},
					"databases_to_backup":        schema.StringAttribute{Optional: true},
					"dump_all":                   schema.BoolAttribute{Optional: true, Computed: true},
					"backup_now":                 schema.BoolAttribute{Optional: true},
					"retain_amount_locally":      schema.Int64Attribute{Optional: true, Computed: true},
					"retain_days_locally":        schema.Int64Attribute{Optional: true, Computed: true},
					"retain_max_storage_locally": schema.Int64Attribute{Optional: true, Computed: true},
					"retain_amount_s3":           schema.Int64Attribute{Optional: true, Computed: true},
					"retain_days_s3":             schema.Int64Attribute{Optional: true, Computed: true},
					"retain_max_storage_s3":      schema.Int64Attribute{Optional: true, Computed: true},
					"timeout":                    schema.Int64Attribute{Optional: true, Computed: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				type v0Model struct {
					ID                    types.Int64  `tfsdk:"id"`
					UUID                  types.String `tfsdk:"uuid"`
					DatabaseUUID          types.String `tfsdk:"database_uuid"`
					Frequency             types.String `tfsdk:"frequency"`
					Enabled               types.Bool   `tfsdk:"enabled"`
					SaveS3                types.Bool   `tfsdk:"save_s3"`
					S3StorageID           types.String `tfsdk:"s3_storage_id"`
					DatabasesToBackup     types.String `tfsdk:"databases_to_backup"`
					DumpAll               types.Bool   `tfsdk:"dump_all"`
					BackupNow             types.Bool   `tfsdk:"backup_now"`
					RetainAmountLocally   types.Int64  `tfsdk:"retain_amount_locally"`
					RetainDaysLocally     types.Int64  `tfsdk:"retain_days_locally"`
					RetainMaxStorageLocal types.Int64  `tfsdk:"retain_max_storage_locally"`
					RetainAmountS3        types.Int64  `tfsdk:"retain_amount_s3"`
					RetainDaysS3          types.Int64  `tfsdk:"retain_days_s3"`
					RetainMaxStorageS3    types.Int64  `tfsdk:"retain_max_storage_s3"`
					Timeout               types.Int64  `tfsdk:"timeout"`
				}
				var old v0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &databaseBackupResourceModel{
					ID: old.ID, UUID: old.UUID, DatabaseUUID: old.DatabaseUUID,
					Frequency: old.Frequency, Enabled: old.Enabled, SaveS3: old.SaveS3,
					S3StorageUUID: old.S3StorageID, DatabasesToBackup: old.DatabasesToBackup,
					DumpAll: old.DumpAll, BackupNow: old.BackupNow,
					RetainAmountLocally: old.RetainAmountLocally, RetainDaysLocally: old.RetainDaysLocally,
					RetainMaxStorageLocal: old.RetainMaxStorageLocal, RetainAmountS3: old.RetainAmountS3,
					RetainDaysS3: old.RetainDaysS3, RetainMaxStorageS3: old.RetainMaxStorageS3,
					Timeout: old.Timeout,
				})...)
			},
		},
	}
}

// readBackup looks up a backup by listing all backups and matching by UUID or
// numeric ID. The individual GET endpoint (/api/v1/databases/{uuid}/backups/{id})
// exists in the client but returns 404 on some Coolify versions, so we use the
// list approach for reliability.
func (r *databaseBackupResource) readBackup(ctx context.Context, dbUUID string, backupID int, uuid types.String) (*client.DatabaseBackup, error) {
	backups, err := r.client.ListDatabaseBackups(ctx, dbUUID)
	if err != nil {
		return nil, err
	}
	for i := range backups {
		if !uuid.IsNull() && !uuid.IsUnknown() && backups[i].UUID == uuid.ValueString() {
			return &backups[i], nil
		}
		if backupID != 0 && backups[i].ID == backupID {
			return &backups[i], nil
		}
	}
	return nil, nil
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
	m.SaveS3 = types.BoolValue(b.SaveS3)
	// The API may return s3_storage_id as a numeric FK, not the UUID the
	// user configured. Only populate on import (when state is empty);
	// otherwise preserve the user's configured value.
	if b.S3StorageID != "" && (m.S3StorageUUID.IsNull() || m.S3StorageUUID.IsUnknown()) {
		m.S3StorageUUID = flex.StringToFramework(b.S3StorageID)
	}
	m.DatabasesToBackup = flex.StringToFramework(b.DatabasesToBackup)
	m.DumpAll = types.BoolValue(b.DumpAll)
	m.RetainAmountLocally = flex.Int64PtrToFramework(b.RetainAmountLocally)
	m.RetainDaysLocally = flex.Int64PtrToFramework(b.RetainDaysLocally)
	m.RetainMaxStorageLocal = flex.Float64PtrToInt64Framework(b.RetainMaxStorageLocal)
	m.RetainAmountS3 = flex.Int64PtrToFramework(b.RetainAmountS3)
	m.RetainDaysS3 = flex.Int64PtrToFramework(b.RetainDaysS3)
	m.RetainMaxStorageS3 = flex.Float64PtrToInt64Framework(b.RetainMaxStorageS3)
	m.Timeout = flex.Int64PtrToFramework(b.Timeout)
}
