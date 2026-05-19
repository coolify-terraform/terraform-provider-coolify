package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	dbcommon "github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &mysqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &mysqlDatabaseResource{}
	_ resource.ResourceWithImportState = &mysqlDatabaseResource{}
)

type mysqlDatabaseResource struct{ client *client.Client }

type mysqlDatabaseResourceModel struct {
	dbcommon.CommonModel
	// Type-specific
	MysqlUser         types.String `tfsdk:"mysql_user"`
	MysqlPassword     types.String `tfsdk:"mysql_password"`
	MysqlDatabase     types.String `tfsdk:"mysql_database"`
	MysqlRootPassword types.String `tfsdk:"mysql_root_password"`
	MysqlConf         types.String `tfsdk:"mysql_conf"`
	EnableSSL         types.Bool   `tfsdk:"enable_ssl"`
	SSLMode           types.String `tfsdk:"ssl_mode"`
}

func NewResource() resource.Resource { return &mysqlDatabaseResource{} }

func (r *mysqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_mysql"
}

func (r *mysqlDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a MySQL database resource on Coolify.",
		Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
			"mysql_user":          schema.StringAttribute{MarkdownDescription: "The MySQL user name (maps to `MYSQL_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_password":      schema.StringAttribute{MarkdownDescription: "The MySQL user password (maps to `MYSQL_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_database":      schema.StringAttribute{MarkdownDescription: "The default database name (maps to `MYSQL_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_root_password": schema.StringAttribute{MarkdownDescription: "The MySQL root password (maps to `MYSQL_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_conf":          schema.StringAttribute{MarkdownDescription: "Custom MySQL configuration (base64-encoded `my.cnf` content).", Optional: true},
			"enable_ssl":          dbcommon.EnableSSLAttr(),
			"ssl_mode":            dbcommon.SSLModeMysqlAttr(),
		}),
	}
}

func (r *mysqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = dbcommon.ConfigureDatabase(req, resp)
}

func (r *mysqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mysqlDatabaseResourceModel
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_mysql"})

	input := client.CreateMysqlInput{ServerUUID: plan.ServerUUID.ValueString(), ProjectUUID: plan.ProjectUUID.ValueString(), EnvironmentName: plan.EnvironmentName.ValueString()}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.MysqlUser, plan.MysqlUser)
	flex.SetIfKnown(&input.MysqlPassword, plan.MysqlPassword)
	flex.SetIfKnown(&input.MysqlDatabase, plan.MysqlDatabase)
	flex.SetIfKnown(&input.MysqlRootPassword, plan.MysqlRootPassword)
	input.IsPublic = flex.BoolValueOrNull(plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)
	created, err := r.client.CreateDatabase(ctx, "mysql", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MySQL database",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	dbcommon.NormalizeCommonCreateState(&plan.CommonModel)
	flex.NormalizeUnknownString(&plan.MysqlUser)
	flex.NormalizeUnknownString(&plan.MysqlPassword)
	flex.NormalizeUnknownString(&plan.MysqlDatabase)
	flex.NormalizeUnknownString(&plan.MysqlRootPassword)
	flex.NormalizeUnknownString(&plan.MysqlConf)
	flex.NormalizeUnknownString(&plan.SSLMode)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := plan.ExtFields().WithSSL(&plan.EnableSSL, &plan.SSLMode)
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if dbcommon.HasExtendedFields(ext) || strSet(plan.MysqlConf) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MysqlConf, plan.MysqlConf)
		if _, err := r.client.UpdateDatabase(ctx, created.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MySQL database extended fields", fmt.Sprintf("MySQL database %s: %s", created.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "MySQL database", created.UUID, err)
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_mysql", "uuid": created.UUID})
}

func (r *mysqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_database_mysql", "uuid": state.UUID.ValueString()})

	db, err := dbcommon.ReadDatabase(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MySQL database", fmt.Sprintf("MySQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
	if db == nil {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_database_mysql", "uuid": state.UUID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	flattenDatabase(db, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mysqlDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_mysql", "uuid": uuid})

	input := client.UpdateDatabaseInput{
		Name:              flex.StringIfChanged(plan.Name, state.Name),
		Description:       flex.StringIfChanged(plan.Description, state.Description),
		Image:             flex.StringIfChanged(plan.Image, state.Image),
		IsPublic:          flex.BoolIfChanged(plan.IsPublic, state.IsPublic),
		PublicPort:        flex.Int64IfChanged(plan.PublicPort, state.PublicPort),
		MysqlUser:         flex.StringIfChanged(plan.MysqlUser, state.MysqlUser),
		MysqlPassword:     flex.StringIfChanged(plan.MysqlPassword, state.MysqlPassword),
		MysqlDatabase:     flex.StringIfChanged(plan.MysqlDatabase, state.MysqlDatabase),
		MysqlRootPassword: flex.StringIfChanged(plan.MysqlRootPassword, state.MysqlRootPassword),
		MysqlConf:         flex.StringIfChanged(plan.MysqlConf, state.MysqlConf),
	}
	dbcommon.SetUpdateExtendedDiff(&input, plan.ExtFields().WithSSL(&plan.EnableSSL, &plan.SSLMode), state.ExtFields().WithSSL(&state.EnableSSL, &state.SSLMode))
	db, err := dbcommon.UpdateDatabase(ctx, r.client, uuid, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MySQL database", fmt.Sprintf("MySQL database %s: %s", uuid, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mysqlDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_database_mysql", "uuid": state.UUID.ValueString()})

	if err := dbcommon.DeleteDatabase(ctx, r.client, "coolify_database_mysql", state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MySQL database", fmt.Sprintf("MySQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *mysqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, req, resp)
}

func flattenDatabase(db *client.Database, m *mysqlDatabaseResourceModel) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields().WithSSL(&m.EnableSSL, &m.SSLMode))
	m.MysqlUser = flex.StringToFramework(db.MysqlUser)
	// Preserve passwords from plan/state when the API hides sensitive fields.
	if db.MysqlPassword != "" {
		m.MysqlPassword = types.StringValue(db.MysqlPassword)
	} else if m.MysqlPassword.IsUnknown() {
		m.MysqlPassword = types.StringNull()
	}
	m.MysqlDatabase = flex.StringToFramework(db.MysqlDatabase)
	if db.MysqlRootPassword != "" {
		m.MysqlRootPassword = types.StringValue(db.MysqlRootPassword)
	} else if m.MysqlRootPassword.IsUnknown() {
		m.MysqlRootPassword = types.StringNull()
	}
	flex.SetStringOrClear(&m.MysqlConf, db.MysqlConf)
}
